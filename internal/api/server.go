// Package api wires ztpanel's REST API: thin HTTP handlers over
// internal/ztapi (the ZeroTier controller) and internal/authsvc
// (ztpanel's own users/sessions).
package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/antonio-damaso/ztpanel/internal/authsvc"
	"github.com/antonio-damaso/ztpanel/internal/ztapi"
)

const sessionCookieName = "ztpanel_session"

type Server struct {
	zt       *ztapi.Client
	users    *authsvc.UserStore
	sessions *authsvc.SessionSigner
	mux      *http.ServeMux
}

func NewServer(zt *ztapi.Client, users *authsvc.UserStore, sessions *authsvc.SessionSigner) *Server {
	s := &Server{zt: zt, users: users, sessions: sessions, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /api/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/logout", s.requireAuth(s.handleLogout))
	s.mux.HandleFunc("GET /api/me", s.requireAuth(s.handleMe))
	s.mux.HandleFunc("POST /api/me/password", s.requireAuth(s.handleChangeOwnPassword))

	s.mux.HandleFunc("GET /api/status", s.requireAuth(s.handleStatus))

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

const userContextKey contextKey = "user"

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		next(w, r.WithContext(ctx))
	}
}

func userFromContext(r *http.Request) string {
	if u, ok := r.Context().Value(userContextKey).(string); ok {
		return u
	}
	return ""
}
