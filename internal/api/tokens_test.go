package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/antonio-damaso/ztpanel/internal/apitoken"
)

func TestTokenLifecycle(t *testing.T) {
	env := newTestEnv(t)
	c := env.loginAs(t, "alice", "alice-password1")

	var created map[string]any
	resp := doJSON(t, c, "POST", env.url("/api/tokens"), map[string]any{"name": "my laptop"})
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("create token: status %d, want 201", resp.StatusCode)
	}
	decodeBody(t, resp, &created)
	plaintext, _ := created["token"].(string)
	if plaintext == "" {
		t.Fatalf("response did not include the plaintext token: %+v", created)
	}

	// The token authenticates /api/me (and anything else requireAuth guards).
	meResp := doBearer(t, plaintext, "GET", env.url("/api/me"), nil)
	defer meResp.Body.Close()
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/me with bearer token: status %d, want 200", meResp.StatusCode)
	}
	var me map[string]string
	decodeBody(t, meResp, &me)
	if me["username"] != "alice" {
		t.Fatalf("me = %+v, want username alice", me)
	}

	// Listing never includes the plaintext or hash.
	var list []map[string]any
	listResp := doJSON(t, c, "GET", env.url("/api/tokens"), nil)
	decodeBody(t, listResp, &list)
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if hash, _ := list[0]["hash"].(string); hash != "" {
		t.Errorf("token listing leaked a non-empty hash: %+v", list[0])
	}
	if _, leaked := list[0]["token"]; leaked {
		t.Errorf("token listing leaked the plaintext: %+v", list[0])
	}

	// Revoking it invalidates the bearer token immediately.
	id, _ := created["id"].(string)
	revokeResp := doJSON(t, c, "DELETE", env.url("/api/tokens/"+id), nil)
	revokeResp.Body.Close()
	if revokeResp.StatusCode != http.StatusOK {
		t.Fatalf("revoke token: status %d, want 200", revokeResp.StatusCode)
	}

	afterResp := doBearer(t, plaintext, "GET", env.url("/api/me"), nil)
	defer afterResp.Body.Close()
	if afterResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /api/me with revoked token: status %d, want 401", afterResp.StatusCode)
	}
}

func TestTokenRejectsBogusBearer(t *testing.T) {
	env := newTestEnv(t)
	resp := doBearer(t, "ztpanel_not-a-real-token", "GET", env.url("/api/me"), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestReadOnlyTokenCanReadButNotWrite(t *testing.T) {
	env := newTestEnv(t)
	c := env.loginAs(t, "alice", "alice-password1")

	var created map[string]any
	resp := doJSON(t, c, "POST", env.url("/api/tokens"), map[string]any{"name": "ci bot", "readOnly": true})
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("create token: status %d, want 201", resp.StatusCode)
	}
	decodeBody(t, resp, &created)
	if ro, _ := created["readOnly"].(bool); !ro {
		t.Fatalf("create response readOnly = %v, want true", created["readOnly"])
	}
	plaintext, _ := created["token"].(string)

	readResp := doBearer(t, plaintext, "GET", env.url("/api/networks"), nil)
	defer readResp.Body.Close()
	if readResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/networks with a read-only token: status %d, want 200", readResp.StatusCode)
	}

	writeResp := doBearer(t, plaintext, "POST", env.url("/api/networks"), map[string]any{"name": "nope"})
	defer writeResp.Body.Close()
	if writeResp.StatusCode != http.StatusForbidden {
		t.Fatalf("POST /api/networks with a read-only token: status %d, want 403", writeResp.StatusCode)
	}
}

func TestExpiredTokenIsRejectedAt401(t *testing.T) {
	env := newTestEnv(t)
	if err := env.users.Create("alice", "alice-password1", false); err != nil {
		t.Fatalf("Create user: %v", err)
	}
	past := time.Now().Add(-time.Hour)
	plaintext, _, err := env.tokens.Create("alice", "stale", apitoken.CreateOptions{ExpiresAt: &past})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	resp := doBearer(t, plaintext, "GET", env.url("/api/me"), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status with an expired token = %d, want 401", resp.StatusCode)
	}
}

func TestRevokeTokenIsScopedToOwner(t *testing.T) {
	env := newTestEnv(t)
	alice := env.loginAs(t, "alice", "alice-password1")
	bob := env.loginAs(t, "bob", "bob-password1")

	var created map[string]any
	resp := doJSON(t, alice, "POST", env.url("/api/tokens"), map[string]any{"name": "alice's token"})
	decodeBody(t, resp, &created)
	id, _ := created["id"].(string)

	stolen := doJSON(t, bob, "DELETE", env.url("/api/tokens/"+id), nil)
	stolen.Body.Close()
	if stolen.StatusCode != http.StatusNotFound {
		t.Fatalf("bob revoking alice's token: status %d, want 404", stolen.StatusCode)
	}
}
