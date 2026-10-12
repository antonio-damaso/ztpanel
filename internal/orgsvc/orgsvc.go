// Package orgsvc adds organizations on top of ztpanel's users: a way
// to group networks and scope who can see/manage them, the way ztnet
// does. The ZeroTier controller itself has no concept of
// organizations, so ztpanel keeps this entirely in its own store and
// maps network IDs to an organization locally.
package orgsvc

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

var (
	ErrNotFound      = errors.New("organization not found")
	ErrAlreadyMember = errors.New("user is already a member")
	ErrNotMember     = errors.New("user is not a member")
	ErrForbidden     = errors.New("insufficient role")
)

type Role string

const (
	RoleAdmin Role = "ADMIN"
	RoleUser  Role = "USER"
)

type Member struct {
	Username string `json:"username"`
	Role     Role   `json:"role"`
}

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Members   []Member  `json:"members"`
	// NetworkIDs are ZeroTier network IDs assigned to this
	// organization. A network not listed in any organization is
	// "personal" -- visible to every ztpanel user, same as before
	// organizations existed.
	NetworkIDs []string `json:"networkIds"`
}

type store struct {
	Organizations map[string]*Organization `json:"organizations"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := s.save(store{Organizations: map[string]*Organization{}}); err != nil {
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
	if st.Organizations == nil {
		st.Organizations = map[string]*Organization{}
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

// ListForUser returns every organization the given user belongs to.
func (s *Store) ListForUser(username string) ([]*Organization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	out := []*Organization{}
	for _, org := range st.Organizations {
		if memberRole(org, username) != "" {
			out = append(out, org)
		}
	}
	return out, nil
}

func (s *Store) Get(id string) (*Organization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	org, ok := st.Organizations[id]
	if !ok {
		return nil, ErrNotFound
	}
	return org, nil
}

// Create makes a new organization with the given user as its first
// (admin) member.
func (s *Store) Create(name, creatorUsername string) (*Organization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return nil, err
	}

	org := &Organization{
		ID:         newID(),
		Name:       name,
		CreatedAt:  time.Now(),
		Members:    []Member{{Username: creatorUsername, Role: RoleAdmin}},
		NetworkIDs: []string{},
	}
	st.Organizations[org.ID] = org
	if err := s.save(st); err != nil {
		return nil, err
	}
	return org, nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return err
	}
	if _, ok := st.Organizations[id]; !ok {
		return ErrNotFound
	}
	delete(st.Organizations, id)
	return s.save(st)
}

// AddMember adds a user to an organization. Only an existing admin of
// that organization may call this (enforced by the caller, which
// knows the acting user).
func (s *Store) AddMember(orgID, username string, role Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return err
	}
	org, ok := st.Organizations[orgID]
	if !ok {
		return ErrNotFound
	}
	if memberRole(org, username) != "" {
		return ErrAlreadyMember
	}
	org.Members = append(org.Members, Member{Username: username, Role: role})
	return s.save(st)
}

func (s *Store) RemoveMember(orgID, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return err
	}
	org, ok := st.Organizations[orgID]
	if !ok {
		return ErrNotFound
	}
	idx := -1
	for i, m := range org.Members {
		if m.Username == username {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ErrNotMember
	}
	org.Members = append(org.Members[:idx], org.Members[idx+1:]...)
	return s.save(st)
}

// AssignNetwork links a ZeroTier network ID to an organization. A
// network may belong to at most one organization at a time.
func (s *Store) AssignNetwork(orgID, networkID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return err
	}
	for _, org := range st.Organizations {
		org.NetworkIDs = removeString(org.NetworkIDs, networkID)
	}
	org, ok := st.Organizations[orgID]
	if !ok {
		return ErrNotFound
	}
	org.NetworkIDs = append(org.NetworkIDs, networkID)
	return s.save(st)
}

// UnassignNetwork removes a network from whichever organization it
// belongs to, making it personal/ungrouped again.
func (s *Store) UnassignNetwork(networkID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return err
	}
	for _, org := range st.Organizations {
		org.NetworkIDs = removeString(org.NetworkIDs, networkID)
	}
	return s.save(st)
}

// AssignedNetworkIDs returns the set of every network ID currently
// assigned to any organization, so callers can filter the "personal /
// ungrouped" network list.
func (s *Store) AssignedNetworkIDs() (map[string]bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, org := range st.Organizations {
		for _, id := range org.NetworkIDs {
			out[id] = true
		}
	}
	return out, nil
}

// OrgForNetwork returns the ID of the organization a network belongs
// to, or "" if it is personal/ungrouped.
func (s *Store) OrgForNetwork(networkID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.load()
	if err != nil {
		return "", err
	}
	for _, org := range st.Organizations {
		for _, id := range org.NetworkIDs {
			if id == networkID {
				return org.ID, nil
			}
		}
	}
	return "", nil
}

// MemberRole returns a user's role in an organization, or "" if they
// are not a member.
func (s *Store) MemberRole(orgID, username string) (Role, error) {
	org, err := s.Get(orgID)
	if err != nil {
		return "", err
	}
	return memberRole(org, username), nil
}

func memberRole(org *Organization, username string) Role {
	for _, m := range org.Members {
		if m.Username == username {
			return m.Role
		}
	}
	return ""
}

func removeString(list []string, value string) []string {
	out := list[:0]
	for _, v := range list {
		if v != value {
			out = append(out, v)
		}
	}
	return out
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
