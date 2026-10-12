package authsvc

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

var ErrSessionInvalid = errors.New("invalid or expired session")

const sessionTTL = 7 * 24 * time.Hour

type sessionClaims struct {
	User    string `json:"user"`
	Expires int64  `json:"exp"`
}

// SessionSigner issues and verifies stateless, HMAC-signed session
// tokens. There is no server-side session table to manage or get out
// of sync: the cookie itself is the session, so a restart of ztpanel
// never logs anyone out, and revocation is simply "rotate the key" or
// "wait for expiry".
type SessionSigner struct {
	key []byte
}

func NewSessionSigner(keyPath string) (*SessionSigner, error) {
	key, err := loadOrCreateKey(keyPath)
	if err != nil {
		return nil, err
	}
	return &SessionSigner{key: key}, nil
}

func loadOrCreateKey(path string) ([]byte, error) {
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		return b, nil
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generating session key: %w", err)
	}
	if err := os.WriteFile(path, key, 0o600); err != nil {
		return nil, fmt.Errorf("persisting session key: %w", err)
	}
	return key, nil
}

func (s *SessionSigner) Issue(username string) (string, time.Time) {
	expires := time.Now().Add(sessionTTL)
	claims := sessionClaims{User: username, Expires: expires.Unix()}
	payload, _ := json.Marshal(claims)

	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	sig := s.sign(payloadB64)
	return payloadB64 + "." + sig, expires
}

func (s *SessionSigner) Verify(token string) (string, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", ErrSessionInvalid
	}
	payloadB64, sig := parts[0], parts[1]

	if !hmac.Equal([]byte(sig), []byte(s.sign(payloadB64))) {
		return "", ErrSessionInvalid
	}

	payload, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", ErrSessionInvalid
	}
	var claims sessionClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", ErrSessionInvalid
	}
	if time.Now().Unix() > claims.Expires {
		return "", ErrSessionInvalid
	}
	return claims.User, nil
}

func (s *SessionSigner) sign(data string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
