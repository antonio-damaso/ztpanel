package authsvc

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestUserStore(t *testing.T) *UserStore {
	t.Helper()
	s, err := NewUserStore(filepath.Join(t.TempDir(), "users.json"))
	if err != nil {
		t.Fatalf("NewUserStore: %v", err)
	}
	return s
}

func TestUserStoreCreateAndAuthenticate(t *testing.T) {
	s := newTestUserStore(t)

	if err := s.Create("alice", "correct-password", false); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := s.Authenticate("alice", "wrong-password"); err != ErrInvalidCredentials {
		t.Fatalf("Authenticate with wrong password: got %v, want ErrInvalidCredentials", err)
	}
	if _, err := s.Authenticate("nobody", "whatever"); err != ErrInvalidCredentials {
		t.Fatalf("Authenticate unknown user: got %v, want ErrInvalidCredentials", err)
	}

	user, err := s.Authenticate("alice", "correct-password")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if user.Name != "alice" {
		t.Errorf("Name = %q, want alice", user.Name)
	}
	if user.PasswordHash != "" {
		t.Errorf("Authenticate leaked PasswordHash: %q", user.PasswordHash)
	}
}

func TestUserStoreAuthenticateUnknownUserCostsRealBcryptWork(t *testing.T) {
	s := newTestUserStore(t)
	if err := s.Create("alice", "correct-password", false); err != nil {
		t.Fatalf("Create: %v", err)
	}

	measure := func(name string) time.Duration {
		start := time.Now()
		if _, err := s.Authenticate(name, "whatever-password"); err != ErrInvalidCredentials {
			t.Fatalf("Authenticate(%q): got %v, want ErrInvalidCredentials", name, err)
		}
		return time.Since(start)
	}

	// A wrong password for a real user and a lookup of a user that
	// doesn't exist at all must do comparable bcrypt work, so neither
	// path is a faster way to probe for valid usernames.
	wrongPassword := measure("alice")
	unknownUser := measure("nobody")

	ratio := float64(unknownUser) / float64(wrongPassword)
	if ratio < 0.5 || ratio > 2 {
		t.Errorf("unknown-user lookup took %v, wrong-password took %v (ratio %.2f) -- want comparable bcrypt cost", unknownUser, wrongPassword, ratio)
	}
}

func TestUserStoreCreateDuplicate(t *testing.T) {
	s := newTestUserStore(t)
	if err := s.Create("alice", "password123", false); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.Create("alice", "another-password", false); err != ErrUserExists {
		t.Fatalf("Create duplicate: got %v, want ErrUserExists", err)
	}
}

func TestUserStoreListNeverLeaksHash(t *testing.T) {
	s := newTestUserStore(t)
	if err := s.Create("alice", "password123", true); err != nil {
		t.Fatalf("Create: %v", err)
	}
	users, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("List returned %d users, want 1", len(users))
	}
	if users[0].PasswordHash != "" {
		t.Errorf("List leaked PasswordHash: %q", users[0].PasswordHash)
	}
	if !users[0].MustChangePassword {
		t.Errorf("MustChangePassword = false, want true")
	}
}

func TestUserStoreSetPassword(t *testing.T) {
	s := newTestUserStore(t)
	if err := s.Create("alice", "old-password", true); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.SetPassword("alice", "new-password", false); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	if _, err := s.Authenticate("alice", "old-password"); err != ErrInvalidCredentials {
		t.Fatalf("old password still works: %v", err)
	}
	user, err := s.Authenticate("alice", "new-password")
	if err != nil {
		t.Fatalf("Authenticate with new password: %v", err)
	}
	if user.MustChangePassword {
		t.Errorf("MustChangePassword = true, want false after SetPassword(..., false)")
	}
	if err := s.SetPassword("nobody", "whatever1", false); err != ErrUserNotFound {
		t.Fatalf("SetPassword unknown user: got %v, want ErrUserNotFound", err)
	}
}

func TestUserStoreDelete(t *testing.T) {
	s := newTestUserStore(t)
	if err := s.Create("alice", "password123", false); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.Delete("alice"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := s.Delete("alice"); err != ErrUserNotFound {
		t.Fatalf("Delete again: got %v, want ErrUserNotFound", err)
	}
	if _, err := s.Authenticate("alice", "password123"); err != ErrInvalidCredentials {
		t.Fatalf("deleted user can still authenticate: %v", err)
	}
}

func TestUserStoreCount(t *testing.T) {
	s := newTestUserStore(t)
	count, err := s.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 0 {
		t.Fatalf("Count = %d, want 0", count)
	}
	if err := s.Create("alice", "password123", false); err != nil {
		t.Fatalf("Create: %v", err)
	}
	count, err = s.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Fatalf("Count = %d, want 1", count)
	}
}
