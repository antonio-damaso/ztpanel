// Package apitoken issues and verifies long-lived Bearer tokens, used
// to authenticate scripts, the REST API, and the MCP endpoint without
// a browser session. A token is shown once, at creation time; only its
// hash is ever persisted, the same way a password is.
package apitoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("token not found")
	ErrInvalid  = errors.New("invalid token")
)

// tokenPrefix makes a ztpanel token recognizable at a glance (in logs,
// in a client config, ...), the way GitHub/Stripe PATs are prefixed.
const tokenPrefix = "ztpanel_"

type Token struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Hash     string `json:"hash"`
	// Last4 is the token's last 4 characters, kept only so a listing
	// can help the user tell two tokens apart without revealing them.
	Last4      string     `json:"last4"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}

type store struct {
	Tokens map[string]*Token `json:"tokens"` // keyed by hash, for O(1) Authenticate
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := s.save(store{Tokens: map[string]*Token{}}); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) load() (store, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return store{}, err
	}
	var st store
	if err := json.Unmarshal(b, &st); err != nil {
		return store{}, err
	}
	if st.Tokens == nil {
		st.Tokens = map[string]*Token{}
	}
	return st, nil
}

func (s *Store) save(st store) error {
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

// Create mints a new token for username and returns its plaintext
// value (shown to the caller exactly once) alongside the stored
// record, which never carries the plaintext.
func (s *Store) Create(username, name string) (plaintext string, token Token, err error) {
	plaintext, err = generateToken()
	if err != nil {
		return "", Token{}, err
	}
	hash := hashToken(plaintext)

	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return "", Token{}, err
	}

	rec := &Token{
		ID:        newID(),
		Name:      name,
		Username:  username,
		Hash:      hash,
		Last4:     plaintext[len(plaintext)-4:],
		CreatedAt: time.Now(),
	}
	st.Tokens[hash] = rec
	if err := s.save(st); err != nil {
		return "", Token{}, err
	}
	out := *rec
	out.Hash = ""
	return plaintext, out, nil
}

// List returns every token belonging to username, without hashes.
func (s *Store) List(username string) ([]Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]Token, 0)
	for _, t := range st.Tokens {
		if t.Username != username {
			continue
		}
		copy := *t
		copy.Hash = ""
		out = append(out, copy)
	}
	return out, nil
}

// Revoke deletes a token by ID, scoped to username so one user cannot
// revoke another's token by guessing an ID.
func (s *Store) Revoke(id, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return err
	}
	for hash, t := range st.Tokens {
		if t.ID == id && t.Username == username {
			delete(st.Tokens, hash)
			return s.save(st)
		}
	}
	return ErrNotFound
}

// Authenticate verifies a presented token and, on success, returns
// the username it was issued to and records the hit as its last use.
func (s *Store) Authenticate(plaintext string) (string, error) {
	if !strings.HasPrefix(plaintext, tokenPrefix) {
		return "", ErrInvalid
	}
	hash := hashToken(plaintext)

	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return "", err
	}
	t, ok := st.Tokens[hash]
	if !ok {
		return "", ErrInvalid
	}
	now := time.Now()
	t.LastUsedAt = &now
	if err := s.save(st); err != nil {
		return "", err
	}
	return t.Username, nil
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return tokenPrefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
