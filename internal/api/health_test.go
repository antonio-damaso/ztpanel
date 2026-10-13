package api

import (
	"net/http"
	"testing"
)

func TestHealthzIsUnauthenticated(t *testing.T) {
	env := newTestEnv(t)
	resp := doJSON(t, &http.Client{}, "GET", env.url("/healthz"), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	decodeBody(t, resp, &body)
	if body["status"] != "ok" {
		t.Fatalf("body = %+v, want status: ok", body)
	}
}
