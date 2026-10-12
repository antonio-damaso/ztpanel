package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/antonio-damaso/ztpanel/internal/apitoken"
	"github.com/antonio-damaso/ztpanel/internal/auditsvc"
	"github.com/antonio-damaso/ztpanel/internal/authsvc"
	"github.com/antonio-damaso/ztpanel/internal/orgsvc"
	"github.com/antonio-damaso/ztpanel/internal/ztapi"
)

// fakeController is a minimal in-memory stand-in for the ZeroTier
// controller's local JSON API, just enough of it for the api package's
// handlers to exercise their full request/response path end to end.
type fakeController struct {
	mu       sync.Mutex
	networks map[string]map[string]any
	members  map[string]map[string]map[string]any // networkID -> addr -> member
	peers    []map[string]any
	nextID   int
}

func newFakeController() *fakeController {
	return &fakeController{
		networks: map[string]map[string]any{},
		members:  map[string]map[string]map[string]any{},
	}
}

func (f *fakeController) start(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"online": true})
	})

	mux.HandleFunc("GET /controller/network", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		ids := make([]string, 0, len(f.networks))
		for id := range f.networks {
			ids = append(ids, id)
		}
		writeJSON(w, http.StatusOK, ids)
	})

	mux.HandleFunc("POST /controller/network/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var fields map[string]any
		_ = json.NewDecoder(r.Body).Decode(&fields)

		f.mu.Lock()
		defer f.mu.Unlock()
		if id == "______" {
			f.nextID++
			id = "testnw" + itoa(f.nextID)
		}
		net, ok := f.networks[id]
		if !ok {
			net = map[string]any{"nwid": id}
			f.members[id] = map[string]map[string]any{}
		}
		for k, v := range fields {
			net[k] = v
		}
		f.networks[id] = net
		writeJSON(w, http.StatusOK, net)
	})

	mux.HandleFunc("GET /controller/network/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		net, ok := f.networks[r.PathValue("id")]
		f.mu.Unlock()
		if !ok {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, net)
	})

	mux.HandleFunc("DELETE /controller/network/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		delete(f.networks, r.PathValue("id"))
		delete(f.members, r.PathValue("id"))
		f.mu.Unlock()
		writeJSON(w, http.StatusOK, nil)
	})

	mux.HandleFunc("GET /controller/network/{id}/member", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		out := map[string]int{}
		for addr := range f.members[r.PathValue("id")] {
			out[addr] = 1
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("GET /controller/network/{id}/member/{addr}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		member, ok := f.members[r.PathValue("id")][r.PathValue("addr")]
		f.mu.Unlock()
		if !ok {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, member)
	})

	mux.HandleFunc("POST /controller/network/{id}/member/{addr}", func(w http.ResponseWriter, r *http.Request) {
		id, addr := r.PathValue("id"), r.PathValue("addr")
		var fields map[string]any
		_ = json.NewDecoder(r.Body).Decode(&fields)

		f.mu.Lock()
		defer f.mu.Unlock()
		if f.members[id] == nil {
			f.members[id] = map[string]map[string]any{}
		}
		member, ok := f.members[id][addr]
		if !ok {
			member = map[string]any{"address": addr}
		}
		for k, v := range fields {
			member[k] = v
		}
		f.members[id][addr] = member
		writeJSON(w, http.StatusOK, member)
	})

	mux.HandleFunc("DELETE /controller/network/{id}/member/{addr}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		delete(f.members[r.PathValue("id")], r.PathValue("addr"))
		f.mu.Unlock()
		writeJSON(w, http.StatusOK, nil)
	})

	mux.HandleFunc("GET /peer", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		writeJSON(w, http.StatusOK, f.peers)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

// testEnv bundles a live Server plus the pieces a test needs to poke
// directly (the fake controller and the audit log).
type testEnv struct {
	server     *Server
	controller *fakeController
	users      *authsvc.UserStore
	audit      *auditsvc.Log
	tokens     *apitoken.Store
	httpServer *httptest.Server
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dir := t.TempDir()

	controller := newFakeController()
	ctrlSrv := controller.start(t)

	tokenPath := filepath.Join(dir, "authtoken.secret")
	if err := os.WriteFile(tokenPath, []byte("test-token"), 0o600); err != nil {
		t.Fatalf("writing fake authtoken: %v", err)
	}
	zt, err := ztapi.New(ctrlSrv.URL, tokenPath)
	if err != nil {
		t.Fatalf("ztapi.New: %v", err)
	}

	users, err := authsvc.NewUserStore(filepath.Join(dir, "users.json"))
	if err != nil {
		t.Fatalf("NewUserStore: %v", err)
	}
	sessions, err := authsvc.NewSessionSigner(filepath.Join(dir, "session.key"))
	if err != nil {
		t.Fatalf("NewSessionSigner: %v", err)
	}
	orgs, err := orgsvc.NewStore(filepath.Join(dir, "organizations.json"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	audit, err := auditsvc.NewLog(filepath.Join(dir, "activity.log"))
	if err != nil {
		t.Fatalf("NewLog: %v", err)
	}
	tokens, err := apitoken.NewStore(filepath.Join(dir, "tokens.json"))
	if err != nil {
		t.Fatalf("apitoken.NewStore: %v", err)
	}

	s := NewServer(zt, users, sessions, orgs, audit, tokens)
	httpServer := httptest.NewServer(s)
	t.Cleanup(httpServer.Close)

	return &testEnv{server: s, controller: controller, users: users, audit: audit, tokens: tokens, httpServer: httpServer}
}

// client returns an http.Client with its own cookie jar, so each
// logged-in "user" in a test keeps its own session.
func (e *testEnv) client(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	return &http.Client{Jar: jar}
}

func (e *testEnv) url(path string) string {
	return e.httpServer.URL + path
}

// loginAs creates a panel user (if needed) and returns an authenticated
// client for it.
func (e *testEnv) loginAs(t *testing.T, username, password string) *http.Client {
	t.Helper()
	if err := e.users.Create(username, password, false); err != nil && err != authsvc.ErrUserExists {
		t.Fatalf("Create user %s: %v", username, err)
	}
	c := e.client(t)
	resp := doJSON(t, c, "POST", e.url("/api/login"), map[string]any{"username": username, "password": password})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login as %s: status %d", username, resp.StatusCode)
	}
	return c
}

// doBearer sends a request authenticated with an API token (as a
// Bearer header) instead of a session cookie, the way an MCP client
// or script would.
func doBearer(t *testing.T, token, method, url string, body any) *http.Response {
	t.Helper()
	resp := doJSONWithHeaders(t, &http.Client{}, method, url, body, map[string]string{
		"Authorization": "Bearer " + token,
	})
	return resp
}

func doJSON(t *testing.T, c *http.Client, method, url string, body any) *http.Response {
	t.Helper()
	return doJSONWithHeaders(t, c, method, url, body, nil)
}

func doJSONWithHeaders(t *testing.T, c *http.Client, method, url string, body any, headers map[string]string) *http.Response {
	t.Helper()
	var req *http.Request
	var err error
	if body != nil {
		b, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			t.Fatalf("json.Marshal: %v", marshalErr)
		}
		req, err = http.NewRequest(method, url, bytes.NewReader(b))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decoding response body: %v", err)
	}
}
