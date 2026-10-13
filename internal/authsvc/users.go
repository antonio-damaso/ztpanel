// Package authsvc handles ztpanel's own users (who can log into the
// panel) and sessions. It is deliberately independent from ZeroTier's
// own identity/auth -- this is just "who may operate this UI".
package authsvc

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

var ErrUserExists = errors.New("user already exists")
var ErrUserNotFound = errors.New("user not found")
var ErrInvalidCredentials = errors.New("invalid credentials")

type User struct {
	Name         string `json:"name"`
	PasswordHash string `json:"passwordHash"`
	// MustChangePassword forces a password reset on next login.
	MustChangePassword bool `json:"mustChangePassword"`
}

// UserStore persists users as a single JSON file. ztpanel's user base
// is expected to be tiny (a handful of admins), so a flat file with a
// mutex is simpler and more auditable than an embedded database.
type UserStore struct {
	path string
	mu   sync.Mutex
}

func NewUserStore(path string) (*UserStore, error) {
	s := &UserStore{path: path}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := s.save(map[string]User{}); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *UserStore) load() (map[string]User, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	var users map[string]User
	if err := json.Unmarshal(b, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (s *UserStore) save(users map[string]User) error {
	b, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

// Count returns how many users exist. Used to decide whether ztpanel
// should bootstrap a first admin account on startup.
func (s *UserStore) Count() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	users, err := s.load()
	if err != nil {
		return 0, err
	}
	return len(users), nil
}

func (s *UserStore) Create(name, password string, mustChangePassword bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	users, err := s.load()
	if err != nil {
		return err
	}
	if _, exists := users[name]; exists {
		return ErrUserExists
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	users[name] = User{Name: name, PasswordHash: hash, MustChangePassword: mustChangePassword}
	return s.save(users)
}

func (s *UserStore) List() ([]User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	users, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]User, 0, len(users))
	for _, u := range users {
		u.PasswordHash = "" // never leak hashes to callers
		out = append(out, u)
	}
	return out, nil
}

func (s *UserStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	users, err := s.load()
	if err != nil {
		return err
	}
	if _, exists := users[name]; !exists {
		return ErrUserNotFound
	}
	delete(users, name)
	return s.save(users)
}

func (s *UserStore) SetPassword(name, password string, mustChangePassword bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	users, err := s.load()
	if err != nil {
		return err
	}
	u, exists := users[name]
	if !exists {
		return ErrUserNotFound
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	u.MustChangePassword = mustChangePassword
	users[name] = u
	return s.save(users)
}

// Authenticate checks a username/password pair and returns the user
// on success. It does not distinguish "no such user" from "wrong
// password" in the returned error, to avoid leaking which usernames
// exist.
func (s *UserStore) Authenticate(name, password string) (User, error) {
	s.mu.Lock()
	users, err := s.load()
	s.mu.Unlock()
	if err != nil {
		return User{}, err
	}
	u, exists := users[name]
	if !exists {
		return User{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return User{}, ErrInvalidCredentials
	}
	u.PasswordHash = ""
	return u, nil
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
