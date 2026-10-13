package apitoken

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(filepath.Join(t.TempDir(), "tokens.json"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func TestCreateAndAuthenticate(t *testing.T) {
	s := newTestStore(t)

	plaintext, token, err := s.Create("alice", "my laptop", CreateOptions{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.HasPrefix(plaintext, "ztpanel_") {
		t.Fatalf("plaintext = %q, want ztpanel_ prefix", plaintext)
	}
	if token.Hash != "" {
		t.Errorf("Create returned a token record with a hash set: %+v", token)
	}
	if token.Last4 != plaintext[len(plaintext)-4:] {
		t.Errorf("Last4 = %q, want %q", token.Last4, plaintext[len(plaintext)-4:])
	}

	username, readOnly, err := s.Authenticate(plaintext)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if username != "alice" {
		t.Errorf("username = %q, want alice", username)
	}
	if readOnly {
		t.Errorf("readOnly = true, want false by default")
	}
}

func TestAuthenticateRejectsUnknownOrMalformedToken(t *testing.T) {
	s := newTestStore(t)
	s.Create("alice", "my laptop", CreateOptions{})

	for _, bad := range []string{"", "not-a-token", "ztpanel_but-not-real"} {
		if _, _, err := s.Authenticate(bad); err != ErrInvalid {
			t.Errorf("Authenticate(%q): got %v, want ErrInvalid", bad, err)
		}
	}
}

func TestAuthenticateRecordsLastUsed(t *testing.T) {
	s := newTestStore(t)
	plaintext, token, _ := s.Create("alice", "my laptop", CreateOptions{})

	before, err := s.List("alice")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if before[0].LastUsedAt != nil {
		t.Fatalf("LastUsedAt set before any use: %+v", before[0])
	}

	if _, _, err := s.Authenticate(plaintext); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	after, err := s.List("alice")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if after[0].ID != token.ID || after[0].LastUsedAt == nil {
		t.Fatalf("after use = %+v, want LastUsedAt set", after[0])
	}
}

func TestListScopedToUsernameAndNeverLeaksHash(t *testing.T) {
	s := newTestStore(t)
	s.Create("alice", "laptop", CreateOptions{})
	s.Create("bob", "phone", CreateOptions{})

	aliceTokens, err := s.List("alice")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(aliceTokens) != 1 || aliceTokens[0].Name != "laptop" {
		t.Fatalf("alice's tokens = %+v", aliceTokens)
	}
	if aliceTokens[0].Hash != "" {
		t.Errorf("List leaked a hash: %+v", aliceTokens[0])
	}
}

func TestRevokeIsScopedToOwner(t *testing.T) {
	s := newTestStore(t)
	plaintext, token, _ := s.Create("alice", "laptop", CreateOptions{})

	if err := s.Revoke(token.ID, "bob"); err != ErrNotFound {
		t.Fatalf("Revoke by a different user: got %v, want ErrNotFound", err)
	}
	// The token must still work.
	if _, _, err := s.Authenticate(plaintext); err != nil {
		t.Fatalf("Authenticate after a cross-user revoke attempt: %v", err)
	}

	if err := s.Revoke(token.ID, "alice"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, _, err := s.Authenticate(plaintext); err != ErrInvalid {
		t.Fatalf("Authenticate after revoke: got %v, want ErrInvalid", err)
	}

	if err := s.Revoke(token.ID, "alice"); err != ErrNotFound {
		t.Fatalf("Revoke again: got %v, want ErrNotFound", err)
	}
}

func TestReadOnlyTokenIsReportedOnAuthenticate(t *testing.T) {
	s := newTestStore(t)
	plaintext, token, err := s.Create("alice", "read-only laptop", CreateOptions{ReadOnly: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !token.ReadOnly {
		t.Errorf("Create returned ReadOnly = false, want true")
	}

	_, readOnly, err := s.Authenticate(plaintext)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if !readOnly {
		t.Errorf("Authenticate returned readOnly = false, want true")
	}
}

func TestExpiredTokenIsRejected(t *testing.T) {
	s := newTestStore(t)
	past := time.Now().Add(-time.Hour)
	plaintext, _, err := s.Create("alice", "stale", CreateOptions{ExpiresAt: &past})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, _, err := s.Authenticate(plaintext); err != ErrExpired {
		t.Fatalf("Authenticate with an expired token: got %v, want ErrExpired", err)
	}
}

func TestUnexpiredTokenStillWorks(t *testing.T) {
	s := newTestStore(t)
	future := time.Now().Add(time.Hour)
	plaintext, _, err := s.Create("alice", "fresh", CreateOptions{ExpiresAt: &future})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, _, err := s.Authenticate(plaintext); err != nil {
		t.Fatalf("Authenticate with an unexpired token: %v", err)
	}
}

func TestEachTokenIsUnique(t *testing.T) {
	s := newTestStore(t)
	a, _, _ := s.Create("alice", "a", CreateOptions{})
	b, _, _ := s.Create("alice", "b", CreateOptions{})
	if a == b {
		t.Fatalf("two tokens had the same plaintext: %q", a)
	}
}
