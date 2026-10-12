package authsvc

import (
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func newTestSessionSigner(t *testing.T) *SessionSigner {
	t.Helper()
	s, err := NewSessionSigner(filepath.Join(t.TempDir(), "session.key"))
	if err != nil {
		t.Fatalf("NewSessionSigner: %v", err)
	}
	return s
}

func TestSessionIssueAndVerify(t *testing.T) {
	s := newTestSessionSigner(t)

	token, expires := s.Issue("alice")
	if token == "" {
		t.Fatal("Issue returned an empty token")
	}
	if !expires.After(time.Now()) {
		t.Fatalf("expires = %v, want a time in the future", expires)
	}

	username, err := s.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if username != "alice" {
		t.Errorf("Verify username = %q, want alice", username)
	}
}

func TestSessionVerifyRejectsTamperedToken(t *testing.T) {
	s := newTestSessionSigner(t)
	token, _ := s.Issue("alice")

	tampered := token[:len(token)-1] + "x"
	if tampered == token {
		t.Fatal("tampering did not change the token")
	}
	if _, err := s.Verify(tampered); err != ErrSessionInvalid {
		t.Fatalf("Verify tampered token: got %v, want ErrSessionInvalid", err)
	}
}

func TestSessionVerifyRejectsForeignSignature(t *testing.T) {
	a := newTestSessionSigner(t)
	b := newTestSessionSigner(t)

	token, _ := a.Issue("alice")
	if _, err := b.Verify(token); err != ErrSessionInvalid {
		t.Fatalf("Verify with a different key: got %v, want ErrSessionInvalid", err)
	}
}

func TestSessionVerifyRejectsMalformedToken(t *testing.T) {
	s := newTestSessionSigner(t)
	for _, tok := range []string{"", "no-dot-here", "a.b.c", "notbase64!.sig"} {
		if _, err := s.Verify(tok); err != ErrSessionInvalid {
			t.Errorf("Verify(%q): got %v, want ErrSessionInvalid", tok, err)
		}
	}
}

func TestSessionVerifyRejectsExpiredToken(t *testing.T) {
	s := newTestSessionSigner(t)

	payload, _ := json.Marshal(sessionClaims{User: "alice", Expires: time.Now().Add(-time.Minute).Unix()})
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	token := payloadB64 + "." + s.sign(payloadB64)

	if _, err := s.Verify(token); err != ErrSessionInvalid {
		t.Fatalf("Verify expired token: got %v, want ErrSessionInvalid", err)
	}
}

func TestSessionKeyPersistsAcrossSigners(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "session.key")
	a, err := NewSessionSigner(keyPath)
	if err != nil {
		t.Fatalf("NewSessionSigner: %v", err)
	}
	token, _ := a.Issue("alice")

	b, err := NewSessionSigner(keyPath)
	if err != nil {
		t.Fatalf("NewSessionSigner (reload): %v", err)
	}
	username, err := b.Verify(token)
	if err != nil {
		t.Fatalf("Verify after reload: %v", err)
	}
	if username != "alice" {
		t.Errorf("username = %q, want alice", username)
	}
}
