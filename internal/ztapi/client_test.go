package ztapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	tokenPath := filepath.Join(t.TempDir(), "authtoken.secret")
	if err := os.WriteFile(tokenPath, []byte("  secret-token  \n"), 0o600); err != nil {
		t.Fatalf("writing fake authtoken: %v", err)
	}

	c, err := New(srv.URL, tokenPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestNewReadsAndTrimsAuthToken(t *testing.T) {
	var gotHeader string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-ZT1-Auth")
		w.Write([]byte(`{"online":true}`))
	})

	if _, err := c.Status(); err != nil {
		t.Fatalf("Status: %v", err)
	}
	if gotHeader != "secret-token" {
		t.Fatalf("X-ZT1-Auth = %q, want %q (trimmed)", gotHeader, "secret-token")
	}
}

func TestNewFailsOnMissingAuthTokenFile(t *testing.T) {
	if _, err := New("http://127.0.0.1:0", filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("New with a missing authtoken file: got nil error, want one")
	}
}

func TestStatus(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/status" {
			t.Errorf("request = %s %s, want GET /status", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"online": true, "version": "1.14.0"})
	})

	status, err := c.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status["online"] != true || status["version"] != "1.14.0" {
		t.Fatalf("status = %+v", status)
	}
}

func TestListNetworkIDs(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/controller/network" {
			t.Errorf("path = %s, want /controller/network", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]string{"abc123", "def456"})
	})

	ids, err := c.ListNetworkIDs()
	if err != nil {
		t.Fatalf("ListNetworkIDs: %v", err)
	}
	if len(ids) != 2 || ids[0] != "abc123" || ids[1] != "def456" {
		t.Fatalf("ids = %v", ids)
	}
}

func TestUpsertNetworkSendsFieldsAsBody(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]any{"nwid": "abc123", "name": gotBody["name"]})
	})

	net, err := c.UpsertNetwork("abc123", map[string]any{"name": "office"})
	if err != nil {
		t.Fatalf("UpsertNetwork: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/controller/network/abc123" {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}
	if gotBody["name"] != "office" {
		t.Fatalf("request body = %+v", gotBody)
	}
	if net["name"] != "office" {
		t.Fatalf("response = %+v", net)
	}
}

func TestDeleteNetwork(t *testing.T) {
	var gotMethod, gotPath string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
	})
	if err := c.DeleteNetwork("abc123"); err != nil {
		t.Fatalf("DeleteNetwork: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/controller/network/abc123" {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}
}

func TestMemberAndPeerEndpoints(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/controller/network/nw1/member":
			json.NewEncoder(w).Encode(map[string]int{"1234567890": 1})
		case r.Method == http.MethodGet && r.URL.Path == "/controller/network/nw1/member/1234567890":
			json.NewEncoder(w).Encode(map[string]any{"address": "1234567890", "authorized": true})
		case r.Method == http.MethodPost && r.URL.Path == "/controller/network/nw1/member/1234567890":
			json.NewEncoder(w).Encode(map[string]any{"address": "1234567890", "authorized": false})
		case r.Method == http.MethodDelete && r.URL.Path == "/controller/network/nw1/member/1234567890":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/peer":
			json.NewEncoder(w).Encode([]map[string]any{{"address": "1234567890", "latency": 5.0}})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	ids, err := c.ListMemberIDs("nw1")
	if err != nil || len(ids) != 1 {
		t.Fatalf("ListMemberIDs: %v, %+v", err, ids)
	}

	member, err := c.GetMember("nw1", "1234567890")
	if err != nil || member["authorized"] != true {
		t.Fatalf("GetMember: %v, %+v", err, member)
	}

	updated, err := c.UpsertMember("nw1", "1234567890", map[string]any{"authorized": false})
	if err != nil || updated["authorized"] != false {
		t.Fatalf("UpsertMember: %v, %+v", err, updated)
	}

	if err := c.DeleteMember("nw1", "1234567890"); err != nil {
		t.Fatalf("DeleteMember: %v", err)
	}

	peers, err := c.Peers()
	if err != nil || len(peers) != 1 || peers[0]["address"] != "1234567890" {
		t.Fatalf("Peers: %v, %+v", err, peers)
	}
}

func TestErrorOnNonOKStatus(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"invalid auth token"}`))
	})

	_, err := c.Status()
	if err == nil {
		t.Fatal("Status with a 403 response: got nil error, want one")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.Status != http.StatusForbidden {
		t.Errorf("apiErr.Status = %d, want 403", apiErr.Status)
	}
	if apiErr.Error() == "" {
		t.Error("APIError.Error() returned an empty string")
	}
}
