package api

import (
	"net/http"
	"testing"
)

func TestLoginWrongPassword(t *testing.T) {
	env := newTestEnv(t)
	if err := env.users.Create("alice", "correct-password", false); err != nil {
		t.Fatalf("Create user: %v", err)
	}

	resp := doJSON(t, env.client(t), "POST", env.url("/api/login"), map[string]any{
		"username": "alice", "password": "wrong-password",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestLoginSuccessSetsCookieAndReturnsFlag(t *testing.T) {
	env := newTestEnv(t)
	if err := env.users.Create("alice", "correct-password", true); err != nil {
		t.Fatalf("Create user: %v", err)
	}

	c := env.client(t)
	resp := doJSON(t, c, "POST", env.url("/api/login"), map[string]any{
		"username": "alice", "password": "correct-password",
	})
	var body struct {
		Username           string `json:"username"`
		MustChangePassword bool   `json:"mustChangePassword"`
	}
	decodeBody(t, resp, &body)
	if body.Username != "alice" || !body.MustChangePassword {
		t.Fatalf("body = %+v, want alice/true", body)
	}

	// The cookie set during login should now authenticate /api/me.
	resp2 := doJSON(t, c, "GET", env.url("/api/me"), nil)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/me after login: status %d", resp2.StatusCode)
	}
}

func TestRequireAuthRejectsMissingCookie(t *testing.T) {
	env := newTestEnv(t)
	resp := doJSON(t, env.client(t), "GET", env.url("/api/me"), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestLogoutClearsSession(t *testing.T) {
	env := newTestEnv(t)
	c := env.loginAs(t, "alice", "correct-password")

	resp := doJSON(t, c, "POST", env.url("/api/logout"), nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("logout status = %d, want 200", resp.StatusCode)
	}

	resp2 := doJSON(t, c, "GET", env.url("/api/me"), nil)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /api/me after logout: status %d, want 401", resp2.StatusCode)
	}
}

func TestChangeOwnPassword(t *testing.T) {
	env := newTestEnv(t)
	c := env.loginAs(t, "alice", "old-password1")

	// Wrong current password is rejected.
	resp := doJSON(t, c, "POST", env.url("/api/me/password"), map[string]any{
		"currentPassword": "not-it", "newPassword": "new-password1",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong current password: status %d, want 401", resp.StatusCode)
	}

	// Too-short new password is rejected.
	resp2 := doJSON(t, c, "POST", env.url("/api/me/password"), map[string]any{
		"currentPassword": "old-password1", "newPassword": "short",
	})
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("short new password: status %d, want 400", resp2.StatusCode)
	}

	// Correct flow actually changes the password.
	resp3 := doJSON(t, c, "POST", env.url("/api/me/password"), map[string]any{
		"currentPassword": "old-password1", "newPassword": "new-password1",
	})
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("change password: status %d, want 200", resp3.StatusCode)
	}

	if _, err := env.users.Authenticate("alice", "new-password1"); err != nil {
		t.Fatalf("Authenticate with new password: %v", err)
	}
}

func TestUsersLifecycle(t *testing.T) {
	env := newTestEnv(t)
	admin := env.loginAs(t, "admin", "admin-password1")

	// Create a second user.
	resp := doJSON(t, admin, "POST", env.url("/api/users"), map[string]any{
		"name": "bob", "password": "bob-password1",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create user: status %d, want 201", resp.StatusCode)
	}

	// Duplicate create fails.
	resp2 := doJSON(t, admin, "POST", env.url("/api/users"), map[string]any{
		"name": "bob", "password": "bob-password1",
	})
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate user: status %d, want 409", resp2.StatusCode)
	}

	// List includes both users, no password hashes.
	var users []map[string]any
	resp3 := doJSON(t, admin, "GET", env.url("/api/users"), nil)
	decodeBody(t, resp3, &users)
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
	for _, u := range users {
		if hash, _ := u["passwordHash"].(string); hash != "" {
			t.Errorf("user listing leaked a non-empty passwordHash: %+v", u)
		}
	}

	// Reset bob's password.
	resp4 := doJSON(t, admin, "POST", env.url("/api/users/bob/password"), map[string]any{
		"password": "bob-new-password1", "mustChangePassword": true,
	})
	resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("reset password: status %d, want 200", resp4.StatusCode)
	}

	// Admin cannot delete themselves.
	resp5 := doJSON(t, admin, "DELETE", env.url("/api/users/admin"), nil)
	resp5.Body.Close()
	if resp5.StatusCode != http.StatusBadRequest {
		t.Fatalf("self-delete: status %d, want 400", resp5.StatusCode)
	}

	// But can delete bob.
	resp6 := doJSON(t, admin, "DELETE", env.url("/api/users/bob"), nil)
	resp6.Body.Close()
	if resp6.StatusCode != http.StatusOK {
		t.Fatalf("delete bob: status %d, want 200", resp6.StatusCode)
	}

	remaining, err := env.users.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, u := range remaining {
		if u.Name == "bob" {
			t.Fatalf("bob still exists after delete")
		}
	}
}
