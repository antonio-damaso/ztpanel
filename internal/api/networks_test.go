package api

import (
	"net/http"
	"testing"
)

func TestNetworksCRUD(t *testing.T) {
	env := newTestEnv(t)
	c := env.loginAs(t, "alice", "alice-password1")

	// Create.
	var created map[string]any
	resp := doJSON(t, c, "POST", env.url("/api/networks"), map[string]any{"name": "office"})
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("create: status %d, want 201", resp.StatusCode)
	}
	decodeBody(t, resp, &created)
	id, _ := created["nwid"].(string)
	if id == "" {
		t.Fatalf("created network has no nwid: %+v", created)
	}
	if created["name"] != "office" {
		t.Errorf("name = %v, want office", created["name"])
	}

	// List should include it.
	var list []map[string]any
	resp2 := doJSON(t, c, "GET", env.url("/api/networks"), nil)
	decodeBody(t, resp2, &list)
	if len(list) != 1 || list[0]["nwid"] != id {
		t.Fatalf("list = %+v, want one network with id %s", list, id)
	}

	// Get by id.
	var got map[string]any
	resp3 := doJSON(t, c, "GET", env.url("/api/networks/"+id), nil)
	decodeBody(t, resp3, &got)
	if got["nwid"] != id {
		t.Fatalf("get = %+v, want nwid %s", got, id)
	}

	// Update.
	resp4 := doJSON(t, c, "PATCH", env.url("/api/networks/"+id), map[string]any{"name": "office-renamed"})
	var updated map[string]any
	decodeBody(t, resp4, &updated)
	if updated["name"] != "office-renamed" {
		t.Fatalf("updated name = %v, want office-renamed", updated["name"])
	}

	// Delete.
	resp5 := doJSON(t, c, "DELETE", env.url("/api/networks/"+id), nil)
	resp5.Body.Close()
	if resp5.StatusCode != http.StatusOK {
		t.Fatalf("delete: status %d, want 200", resp5.StatusCode)
	}
	resp6 := doJSON(t, c, "GET", env.url("/api/networks/"+id), nil)
	resp6.Body.Close()
	if resp6.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete: status %d, want 404", resp6.StatusCode)
	}

	// Every mutation should have left an audit trail.
	entries, err := env.audit.Recent(10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != 3 { // created, updated, deleted
		t.Fatalf("len(audit entries) = %d, want 3: %+v", len(entries), entries)
	}
}

func TestListNetworksExcludesOrgAssigned(t *testing.T) {
	env := newTestEnv(t)
	c := env.loginAs(t, "alice", "alice-password1")

	var net1, net2 map[string]any
	resp1 := doJSON(t, c, "POST", env.url("/api/networks"), map[string]any{"name": "personal"})
	decodeBody(t, resp1, &net1)
	resp2 := doJSON(t, c, "POST", env.url("/api/networks"), map[string]any{"name": "org-owned"})
	decodeBody(t, resp2, &net2)

	var org map[string]any
	resp3 := doJSON(t, c, "POST", env.url("/api/organizations"), map[string]any{"name": "Acme"})
	decodeBody(t, resp3, &org)
	orgID, _ := org["id"].(string)

	net2ID, _ := net2["nwid"].(string)
	resp4 := doJSON(t, c, "POST", env.url("/api/organizations/"+orgID+"/networks/"+net2ID), nil)
	resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("assign network to org: status %d, want 200", resp4.StatusCode)
	}

	var list []map[string]any
	resp5 := doJSON(t, c, "GET", env.url("/api/networks"), nil)
	decodeBody(t, resp5, &list)
	if len(list) != 1 || list[0]["nwid"] != net1["nwid"] {
		t.Fatalf("personal network list = %+v, want only %v", list, net1["nwid"])
	}

	var orgNetworks []map[string]any
	resp6 := doJSON(t, c, "GET", env.url("/api/organizations/"+orgID+"/networks"), nil)
	decodeBody(t, resp6, &orgNetworks)
	if len(orgNetworks) != 1 || orgNetworks[0]["nwid"] != net2ID {
		t.Fatalf("org network list = %+v, want only %v", orgNetworks, net2ID)
	}
}

func TestMembersListEnrichesWithPeerStatus(t *testing.T) {
	env := newTestEnv(t)
	c := env.loginAs(t, "alice", "alice-password1")

	var net map[string]any
	resp := doJSON(t, c, "POST", env.url("/api/networks"), map[string]any{"name": "office"})
	decodeBody(t, resp, &net)
	nwid, _ := net["nwid"].(string)

	resp2 := doJSON(t, c, "PATCH", env.url("/api/networks/"+nwid+"/members/1234567890"), map[string]any{
		"authorized": true,
	})
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("upsert member: status %d, want 200", resp2.StatusCode)
	}

	env.controller.mu.Lock()
	env.controller.peers = []map[string]any{{"address": "1234567890", "latency": 12.0}}
	env.controller.mu.Unlock()

	var members []map[string]any
	resp3 := doJSON(t, c, "GET", env.url("/api/networks/"+nwid+"/members"), nil)
	decodeBody(t, resp3, &members)
	if len(members) != 1 {
		t.Fatalf("len(members) = %d, want 1", len(members))
	}
	peer, ok := members[0]["peer"].(map[string]any)
	if !ok {
		t.Fatalf("member missing enriched peer data: %+v", members[0])
	}
	if peer["latency"] != 12.0 {
		t.Errorf("peer latency = %v, want 12", peer["latency"])
	}

	// Delete the member.
	resp4 := doJSON(t, c, "DELETE", env.url("/api/networks/"+nwid+"/members/1234567890"), nil)
	resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("delete member: status %d, want 200", resp4.StatusCode)
	}
	resp5 := doJSON(t, c, "GET", env.url("/api/networks/"+nwid+"/members/1234567890"), nil)
	resp5.Body.Close()
	if resp5.StatusCode != http.StatusNotFound {
		t.Fatalf("get member after delete: status %d, want 404", resp5.StatusCode)
	}
}

func TestStatusProxiesController(t *testing.T) {
	env := newTestEnv(t)
	c := env.loginAs(t, "alice", "alice-password1")

	var status map[string]any
	resp := doJSON(t, c, "GET", env.url("/api/status"), nil)
	decodeBody(t, resp, &status)
	if status["online"] != true {
		t.Fatalf("status = %+v, want online: true", status)
	}
}
