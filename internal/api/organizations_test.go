package api

import (
	"net/http"
	"testing"
)

func createOrg(t *testing.T, env *testEnv, c *http.Client, name string) map[string]any {
	t.Helper()
	var org map[string]any
	resp := doJSON(t, c, "POST", env.url("/api/organizations"), map[string]any{"name": name})
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("create organization: status %d", resp.StatusCode)
	}
	decodeBody(t, resp, &org)
	return org
}

func TestOrganizationLifecycle(t *testing.T) {
	env := newTestEnv(t)
	admin := env.loginAs(t, "alice", "alice-password1")

	org := createOrg(t, env, admin, "Acme")
	orgID, _ := org["id"].(string)
	if orgID == "" {
		t.Fatalf("org has no id: %+v", org)
	}

	// Creator can read it back.
	var got map[string]any
	resp := doJSON(t, admin, "GET", env.url("/api/organizations/"+orgID), nil)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("get organization: status %d", resp.StatusCode)
	}
	decodeBody(t, resp, &got)
	if got["name"] != "Acme" {
		t.Errorf("name = %v, want Acme", got["name"])
	}

	// It shows up in the creator's list.
	var list []map[string]any
	resp2 := doJSON(t, admin, "GET", env.url("/api/organizations"), nil)
	decodeBody(t, resp2, &list)
	if len(list) != 1 || list[0]["id"] != orgID {
		t.Fatalf("list = %+v, want only %s", list, orgID)
	}
}

func TestOrganizationGetForbiddenForNonMember(t *testing.T) {
	env := newTestEnv(t)
	admin := env.loginAs(t, "alice", "alice-password1")
	outsider := env.loginAs(t, "mallory", "mallory-password1")

	org := createOrg(t, env, admin, "Acme")
	orgID, _ := org["id"].(string)

	resp := doJSON(t, outsider, "GET", env.url("/api/organizations/"+orgID), nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-member get: status %d, want 403", resp.StatusCode)
	}
}

func TestOrganizationMemberManagement(t *testing.T) {
	env := newTestEnv(t)
	admin := env.loginAs(t, "alice", "alice-password1")
	member := env.loginAs(t, "bob", "bob-password1")

	org := createOrg(t, env, admin, "Acme")
	orgID, _ := org["id"].(string)

	// A regular (non-admin, non-member) user cannot add members.
	resp := doJSON(t, member, "POST", env.url("/api/organizations/"+orgID+"/members"), map[string]any{
		"username": "bob", "role": "USER",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-member add member: status %d, want 403", resp.StatusCode)
	}

	// The admin adds bob as a regular member.
	resp2 := doJSON(t, admin, "POST", env.url("/api/organizations/"+orgID+"/members"), map[string]any{
		"username": "bob", "role": "USER",
	})
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("admin add member: status %d, want 200", resp2.StatusCode)
	}

	// Now bob can see the org, but cannot add members (not an admin) or delete it.
	resp3 := doJSON(t, member, "GET", env.url("/api/organizations/"+orgID), nil)
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("bob gets org after being added: status %d, want 200", resp3.StatusCode)
	}

	resp4 := doJSON(t, member, "POST", env.url("/api/organizations/"+orgID+"/members"), map[string]any{
		"username": "carol", "role": "USER",
	})
	resp4.Body.Close()
	if resp4.StatusCode != http.StatusForbidden {
		t.Fatalf("regular member add member: status %d, want 403", resp4.StatusCode)
	}

	resp5 := doJSON(t, member, "DELETE", env.url("/api/organizations/"+orgID), nil)
	resp5.Body.Close()
	if resp5.StatusCode != http.StatusForbidden {
		t.Fatalf("regular member delete org: status %d, want 403", resp5.StatusCode)
	}

	// The admin removes bob.
	resp6 := doJSON(t, admin, "DELETE", env.url("/api/organizations/"+orgID+"/members/bob"), nil)
	resp6.Body.Close()
	if resp6.StatusCode != http.StatusOK {
		t.Fatalf("admin remove member: status %d, want 200", resp6.StatusCode)
	}

	// bob lost access.
	resp7 := doJSON(t, member, "GET", env.url("/api/organizations/"+orgID), nil)
	resp7.Body.Close()
	if resp7.StatusCode != http.StatusForbidden {
		t.Fatalf("bob after removal: status %d, want 403", resp7.StatusCode)
	}

	// Finally the admin deletes the organization.
	resp8 := doJSON(t, admin, "DELETE", env.url("/api/organizations/"+orgID), nil)
	resp8.Body.Close()
	if resp8.StatusCode != http.StatusOK {
		t.Fatalf("admin delete org: status %d, want 200", resp8.StatusCode)
	}
}

func TestAssignAndUnassignNetworkRequiresOrgAdmin(t *testing.T) {
	env := newTestEnv(t)
	admin := env.loginAs(t, "alice", "alice-password1")
	member := env.loginAs(t, "bob", "bob-password1")

	org := createOrg(t, env, admin, "Acme")
	orgID, _ := org["id"].(string)

	resp0 := doJSON(t, admin, "POST", env.url("/api/organizations/"+orgID+"/members"), map[string]any{
		"username": "bob", "role": "USER",
	})
	resp0.Body.Close()

	var net map[string]any
	resp1 := doJSON(t, admin, "POST", env.url("/api/networks"), map[string]any{"name": "office"})
	decodeBody(t, resp1, &net)
	nwid, _ := net["nwid"].(string)

	// Regular member cannot assign.
	resp2 := doJSON(t, member, "POST", env.url("/api/organizations/"+orgID+"/networks/"+nwid), nil)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusForbidden {
		t.Fatalf("member assign network: status %d, want 403", resp2.StatusCode)
	}

	// Admin can assign.
	resp3 := doJSON(t, admin, "POST", env.url("/api/organizations/"+orgID+"/networks/"+nwid), nil)
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("admin assign network: status %d, want 200", resp3.StatusCode)
	}

	var orgNetworks []map[string]any
	resp4 := doJSON(t, admin, "GET", env.url("/api/organizations/"+orgID+"/networks"), nil)
	decodeBody(t, resp4, &orgNetworks)
	if len(orgNetworks) != 1 || orgNetworks[0]["nwid"] != nwid {
		t.Fatalf("org networks = %+v, want only %s", orgNetworks, nwid)
	}

	// Admin can unassign.
	resp5 := doJSON(t, admin, "DELETE", env.url("/api/organizations/"+orgID+"/networks/"+nwid), nil)
	resp5.Body.Close()
	if resp5.StatusCode != http.StatusOK {
		t.Fatalf("admin unassign network: status %d, want 200", resp5.StatusCode)
	}

	var afterUnassign []map[string]any
	resp6 := doJSON(t, admin, "GET", env.url("/api/organizations/"+orgID+"/networks"), nil)
	decodeBody(t, resp6, &afterUnassign)
	if len(afterUnassign) != 0 {
		t.Fatalf("org networks after unassign = %+v, want empty", afterUnassign)
	}
}
