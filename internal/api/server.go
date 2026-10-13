// Package api wires ztpanel's REST API: thin HTTP handlers over
// internal/ztapi (the ZeroTier controller) and internal/authsvc
// (ztpanel's own users/sessions).
package api

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/antonio-damaso/ztpanel/internal/apitoken"
	"github.com/antonio-damaso/ztpanel/internal/auditsvc"
	"github.com/antonio-damaso/ztpanel/internal/authsvc"
	"github.com/antonio-damaso/ztpanel/internal/orgsvc"
	"github.com/antonio-damaso/ztpanel/internal/ratelimit"
	"github.com/antonio-damaso/ztpanel/internal/ztapi"
)

const sessionCookieName = "ztpanel_session"

// loginRateLimit and setupRateLimit bound brute-force attempts against
// the two unauthenticated credential-checking endpoints.
const (
	loginRateLimit  = 10
	setupRateLimit  = 10
	rateLimitWindow = time.Minute
)

type Server struct {
	zt           *ztapi.Client
	users        *authsvc.UserStore
	sessions     *authsvc.SessionSigner
	orgs         *orgsvc.Store
	audit        *auditsvc.Log
	tokens       *apitoken.Store
	loginLimiter *ratelimit.Limiter
	setupLimiter *ratelimit.Limiter
	mux          *http.ServeMux
}

func NewServer(zt *ztapi.Client, users *authsvc.UserStore, sessions *authsvc.SessionSigner, orgs *orgsvc.Store, audit *auditsvc.Log, tokens *apitoken.Store) *Server {
	s := &Server{
		zt: zt, users: users, sessions: sessions, orgs: orgs, audit: audit, tokens: tokens,
		loginLimiter: ratelimit.New(loginRateLimit, rateLimitWindow),
		setupLimiter: ratelimit.New(setupRateLimit, rateLimitWindow),
		mux:          http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)

	s.mux.HandleFunc("GET /api/setup/status", s.handleSetupStatus)
	s.mux.HandleFunc("POST /api/setup", rateLimited(s.setupLimiter, s.handleSetup))

	s.mux.HandleFunc("POST /api/login", rateLimited(s.loginLimiter, s.handleLogin))
	s.mux.HandleFunc("POST /api/logout", s.requireAuth(s.handleLogout))
	s.mux.HandleFunc("GET /api/me", s.requireAuth(s.handleMe))
	s.mux.HandleFunc("POST /api/me/password", s.requireAuth(s.handleChangeOwnPassword))

	s.mux.HandleFunc("GET /api/status", s.requireAuth(s.handleStatus))
	s.mux.HandleFunc("GET /api/activity", s.requireAuth(s.handleActivity))

	s.mux.HandleFunc("GET /api/networks", s.requireAuth(s.handleListNetworks))
	s.mux.HandleFunc("POST /api/networks", s.requireAuth(s.handleCreateNetwork))
	s.mux.HandleFunc("GET /api/networks/{id}", s.requireAuth(s.handleGetNetwork))
	s.mux.HandleFunc("PATCH /api/networks/{id}", s.requireAuth(s.handleUpdateNetwork))
	s.mux.HandleFunc("DELETE /api/networks/{id}", s.requireAuth(s.handleDeleteNetwork))

	s.mux.HandleFunc("GET /api/networks/{id}/members", s.requireAuth(s.handleListMembers))
	s.mux.HandleFunc("GET /api/networks/{id}/members/{addr}", s.requireAuth(s.handleGetMember))
	s.mux.HandleFunc("PATCH /api/networks/{id}/members/{addr}", s.requireAuth(s.handleUpdateMember))
	s.mux.HandleFunc("DELETE /api/networks/{id}/members/{addr}", s.requireAuth(s.handleDeleteMember))

	s.mux.HandleFunc("GET /api/users", s.requireAuth(s.handleListUsers))
	s.mux.HandleFunc("POST /api/users", s.requireAuth(s.handleCreateUser))
	s.mux.HandleFunc("POST /api/users/{name}/password", s.requireAuth(s.handleSetUserPassword))
	s.mux.HandleFunc("DELETE /api/users/{name}", s.requireAuth(s.handleDeleteUser))

	s.mux.HandleFunc("GET /api/organizations", s.requireAuth(s.handleListOrganizations))
	s.mux.HandleFunc("POST /api/organizations", s.requireAuth(s.handleCreateOrganization))
	s.mux.HandleFunc("GET /api/organizations/{orgId}", s.requireAuth(s.handleGetOrganization))
	s.mux.HandleFunc("DELETE /api/organizations/{orgId}", s.requireAuth(s.handleDeleteOrganization))
	s.mux.HandleFunc("POST /api/organizations/{orgId}/members", s.requireAuth(s.handleAddOrgMember))
	s.mux.HandleFunc("DELETE /api/organizations/{orgId}/members/{username}", s.requireAuth(s.handleRemoveOrgMember))
	s.mux.HandleFunc("GET /api/organizations/{orgId}/networks", s.requireAuth(s.handleListOrgNetworks))
	s.mux.HandleFunc("POST /api/organizations/{orgId}/networks/{nwid}", s.requireAuth(s.handleAssignNetworkToOrg))
	s.mux.HandleFunc("DELETE /api/organizations/{orgId}/networks/{nwid}", s.requireAuth(s.handleUnassignNetworkFromOrg))

	s.mux.HandleFunc("GET /api/tokens", s.requireAuth(s.handleListTokens))
	s.mux.HandleFunc("POST /api/tokens", s.requireAuth(s.handleCreateToken))
	s.mux.HandleFunc("DELETE /api/tokens/{id}", s.requireAuth(s.handleRevokeToken))

	s.mux.HandleFunc("POST /mcp", s.requireAuth(s.handleMCP))
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			log.Printf("writeJSON: %v", err)
		}
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

type contextKey string

const (
	userContextKey     contextKey = "user"
	readOnlyContextKey contextKey = "readOnly"
)

// requireAuth accepts either the browser's session cookie or a
// long-lived "Authorization: Bearer <token>" API token (internal/apitoken) --
// the latter is how scripts, the REST API from the CLI, and the /mcp
// endpoint authenticate without a browser.
//
// A read-only API token may call any REST GET/HEAD route, but is
// rejected outright on any other method under /api/*. /mcp is always
// POST, so its own tools are instead gated per tool name in mcp.go.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if bearer, ok := bearerToken(r); ok {
			username, readOnly, err := s.tokens.Authenticate(bearer)
			if err != nil {
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeError(w, http.StatusUnauthorized, "invalid or expired API token")
				return
			}
			if readOnly && strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodGet && r.Method != http.MethodHead {
				writeError(w, http.StatusForbidden, "read-only API token cannot perform this action")
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey, username)
			ctx = context.WithValue(ctx, readOnlyContextKey, readOnly)
			next(w, r.WithContext(ctx))
			return
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		username, err := s.sessions.Verify(cookie.Value)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "session expired")
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, username)
		ctx = context.WithValue(ctx, readOnlyContextKey, false)
		next(w, r.WithContext(ctx))
	}
}

func bearerToken(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return "", false
	}
	return strings.TrimPrefix(auth, prefix), true
}

func userFromContext(r *http.Request) string {
	if u, ok := r.Context().Value(userContextKey).(string); ok {
		return u
	}
	return ""
}

// readOnlyFromContext reports whether the caller authenticated with a
// read-only API token. A session-cookie caller is never read-only.
func readOnlyFromContext(r *http.Request) bool {
	ro, _ := r.Context().Value(readOnlyContextKey).(bool)
	return ro
}

// clientIP returns the request's remote address without its port, for
// use as a rate-limiter key. It does not consult X-Forwarded-For --
// ztpanel is meant to sit directly behind a TLS-terminating reverse
// proxy the deployer controls, not to trust a client-supplied header.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// rateLimited wraps a handler so it returns 429 once its limiter's
// per-IP quota is exhausted, instead of running the handler at all.
func rateLimited(limiter *ratelimit.Limiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow(clientIP(r)) {
			writeError(w, http.StatusTooManyRequests, "too many attempts, try again later")
			return
		}
		next(w, r)
	}
}
