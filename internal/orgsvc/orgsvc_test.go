package orgsvc

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(filepath.Join(t.TempDir(), "organizations.json"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func TestCreateMakesCreatorAdmin(t *testing.T) {
	s := newTestStore(t)
	org, err := s.Create("Acme", "alice")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if org.ID == "" {
		t.Fatal("Create returned an empty ID")
	}
	role, err := s.MemberRole(org.ID, "alice")
	if err != nil {
		t.Fatalf("MemberRole: %v", err)
	}
	if role != RoleAdmin {
		t.Errorf("creator role = %q, want ADMIN", role)
	}
}

func TestListForUser(t *testing.T) {
	s := newTestStore(t)
	orgA, _ := s.Create("Acme", "alice")
	_, _ = s.Create("Globex", "bob")

	orgs, err := s.ListForUser("alice")
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(orgs) != 1 || orgs[0].ID != orgA.ID {
		t.Fatalf("ListForUser(alice) = %+v, want only %s", orgs, orgA.ID)
	}

	orgs, err = s.ListForUser("nobody")
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(orgs) != 0 {
		t.Fatalf("ListForUser(nobody) = %+v, want empty", orgs)
	}
}

func TestAddAndRemoveMember(t *testing.T) {
	s := newTestStore(t)
	org, _ := s.Create("Acme", "alice")

	if err := s.AddMember(org.ID, "bob", RoleUser); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	role, err := s.MemberRole(org.ID, "bob")
	if err != nil || role != RoleUser {
		t.Fatalf("MemberRole(bob) = %q, %v, want USER, nil", role, err)
	}

	if err := s.AddMember(org.ID, "bob", RoleUser); err != ErrAlreadyMember {
		t.Fatalf("AddMember duplicate: got %v, want ErrAlreadyMember", err)
	}

	if err := s.RemoveMember(org.ID, "bob"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	role, err = s.MemberRole(org.ID, "bob")
	if err != nil {
		t.Fatalf("MemberRole after removal: %v", err)
	}
	if role != "" {
		t.Errorf("role after removal = %q, want empty", role)
	}

	if err := s.RemoveMember(org.ID, "bob"); err != ErrNotMember {
		t.Fatalf("RemoveMember again: got %v, want ErrNotMember", err)
	}
}

func TestAddMemberUnknownOrg(t *testing.T) {
	s := newTestStore(t)
	if err := s.AddMember("does-not-exist", "bob", RoleUser); err != ErrNotFound {
		t.Fatalf("AddMember on unknown org: got %v, want ErrNotFound", err)
	}
}

func TestAssignNetworkMovesBetweenOrgs(t *testing.T) {
	s := newTestStore(t)
	orgA, _ := s.Create("Acme", "alice")
	orgB, _ := s.Create("Globex", "bob")

	if err := s.AssignNetwork(orgA.ID, "nwid1"); err != nil {
		t.Fatalf("AssignNetwork: %v", err)
	}
	assigned, err := s.AssignedNetworkIDs()
	if err != nil {
		t.Fatalf("AssignedNetworkIDs: %v", err)
	}
	if !assigned["nwid1"] {
		t.Fatal("nwid1 not marked as assigned")
	}
	orgID, err := s.OrgForNetwork("nwid1")
	if err != nil || orgID != orgA.ID {
		t.Fatalf("OrgForNetwork = %q, %v, want %s, nil", orgID, err, orgA.ID)
	}

	// A network may belong to only one organization: assigning it
	// elsewhere must detach it from orgA.
	if err := s.AssignNetwork(orgB.ID, "nwid1"); err != nil {
		t.Fatalf("AssignNetwork (move): %v", err)
	}
	orgID, err = s.OrgForNetwork("nwid1")
	if err != nil || orgID != orgB.ID {
		t.Fatalf("OrgForNetwork after move = %q, %v, want %s, nil", orgID, err, orgB.ID)
	}

	fresh, err := s.Get(orgA.ID)
	if err != nil {
		t.Fatalf("Get(orgA): %v", err)
	}
	for _, id := range fresh.NetworkIDs {
		if id == "nwid1" {
			t.Fatalf("orgA still lists nwid1 after it moved to orgB: %v", fresh.NetworkIDs)
		}
	}
}

func TestUnassignNetwork(t *testing.T) {
	s := newTestStore(t)
	org, _ := s.Create("Acme", "alice")
	if err := s.AssignNetwork(org.ID, "nwid1"); err != nil {
		t.Fatalf("AssignNetwork: %v", err)
	}
	if err := s.UnassignNetwork("nwid1"); err != nil {
		t.Fatalf("UnassignNetwork: %v", err)
	}
	assigned, err := s.AssignedNetworkIDs()
	if err != nil {
		t.Fatalf("AssignedNetworkIDs: %v", err)
	}
	if assigned["nwid1"] {
		t.Fatal("nwid1 still marked as assigned after UnassignNetwork")
	}
}

func TestDeleteOrganization(t *testing.T) {
	s := newTestStore(t)
	org, _ := s.Create("Acme", "alice")
	if err := s.Delete(org.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(org.ID); err != ErrNotFound {
		t.Fatalf("Get after delete: got %v, want ErrNotFound", err)
	}
	if err := s.Delete(org.ID); err != ErrNotFound {
		t.Fatalf("Delete again: got %v, want ErrNotFound", err)
	}
}

func TestMemberRoleUnknownOrg(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.MemberRole("does-not-exist", "alice"); err != ErrNotFound {
		t.Fatalf("MemberRole on unknown org: got %v, want ErrNotFound", err)
	}
}
