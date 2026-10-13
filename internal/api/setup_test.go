package api

import (
	"net/http"
	"testing"
)

func TestSetupStatusReportsWhetherSetupIsNeeded(t *testing.T) {
	env := newTestEnv(t)

	resp := doJSON(t, env.client(t), "GET", env.url("/api/setup/status"), nil)
	var status map[string]any
	decodeBody(t, resp, &status)
	if status["needsSetup"] != true {
		t.Fatalf("status = %+v, want needsSetup: true on a fresh install", status)
	}
	if _, ok := status["controllerReachable"]; !ok {
		t.Errorf("status = %+v, want a controllerReachable field while setup is pending", status)
	}

	if err := env.users.Create("admin", "admin-password1", false); err != nil {
		t.Fatalf("Create: %v", err)
	}

	resp2 := doJSON(t, env.client(t), "GET", env.url("/api/setup/status"), nil)
	var status2 map[string]any
	decodeBody(t, resp2, &status2)
	if status2["needsSetup"] != false {
		t.Fatalf("status after a user exists = %+v, want needsSetup: false", status2)
	}
	if _, leaked := status2["controllerReachable"]; leaked {
		t.Errorf("status after setup leaked controller info: %+v", status2)
	}
}

func TestSetupCreatesAdminAndLogsIn(t *testing.T) {
	env := newTestEnv(t)
	c := env.client(t)

	resp := doJSON(t, c, "POST", env.url("/api/setup"), map[string]any{
		"username": "alice", "password": "alice-password1",
	})
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("setup: status %d, want 201", resp.StatusCode)
	}
	var body map[string]any
	decodeBody(t, resp, &body)
	if body["username"] != "alice" {
		t.Fatalf("body = %+v, want username alice", body)
	}

	// The response's cookie must already authenticate the caller --
	// no separate login step needed.
	meResp := doJSON(t, c, "GET", env.url("/api/me"), nil)
	defer meResp.Body.Close()
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/me right after setup: status %d, want 200", meResp.StatusCode)
	}

	// And unlike a normal admin-created user, they are not forced to
	// change the password they just chose themselves.
	user, err := env.users.Authenticate("alice", "alice-password1")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if user.MustChangePassword {
		t.Errorf("MustChangePassword = true, want false for a self-chosen setup password")
	}
}

func TestSetupRejectsShortPasswordOrEmptyUsername(t *testing.T) {
	env := newTestEnv(t)
	c := env.client(t)

	resp := doJSON(t, c, "POST", env.url("/api/setup"), map[string]any{"username": "alice", "password": "short"})
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("short password: status %d, want 400", resp.StatusCode)
	}

	resp2 := doJSON(t, c, "POST", env.url("/api/setup"), map[string]any{"username": "", "password": "longenoughpassword"})
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty username: status %d, want 400", resp2.StatusCode)
	}
}

func TestSetupIsRateLimitedPerIP(t *testing.T) {
	env := newTestEnv(t)
	c := env.client(t)

	for i := 0; i < setupRateLimit; i++ {
		resp := doJSON(t, c, "POST", env.url("/api/setup"), map[string]any{"username": "alice", "password": "short"})
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("attempt %d: status %d, want 400", i, resp.StatusCode)
		}
	}

	resp := doJSON(t, c, "POST", env.url("/api/setup"), map[string]any{"username": "alice", "password": "alice-password1"})
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status after exhausting the limit = %d, want 429", resp.StatusCode)
	}
}

func TestSetupRefusesOnceConfigured(t *testing.T) {
	env := newTestEnv(t)
	c := env.client(t)

	resp := doJSON(t, c, "POST", env.url("/api/setup"), map[string]any{"username": "alice", "password": "alice-password1"})
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first setup: status %d, want 201", resp.StatusCode)
	}

	resp2 := doJSON(t, env.client(t), "POST", env.url("/api/setup"), map[string]any{"username": "mallory", "password": "mallory-password1"})
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("second setup attempt: status %d, want 409", resp2.StatusCode)
	}

	// mallory must not have been created.
	if _, err := env.users.Authenticate("mallory", "mallory-password1"); err == nil {
		t.Fatal("a second setup call was able to create a user")
	}
}
