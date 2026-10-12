package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

// mcpTokenEnv returns a testEnv plus a ready-to-use API token for
// "alice", the way an MCP client would authenticate.
func mcpTokenEnv(t *testing.T) (*testEnv, string) {
	t.Helper()
	env := newTestEnv(t)
	c := env.loginAs(t, "alice", "alice-password1")

	var created map[string]any
	resp := doJSON(t, c, "POST", env.url("/api/tokens"), map[string]any{"name": "mcp client"})
	decodeBody(t, resp, &created)
	token, _ := created["token"].(string)
	if token == "" {
		t.Fatalf("could not mint a token for the MCP tests: %+v", created)
	}
	return env, token
}

func mcpCall(t *testing.T, env *testEnv, token string, id int, method string, params any) map[string]any {
	t.Helper()
	body := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		body["params"] = params
	}
	resp := doBearer(t, token, "POST", env.url("/mcp"), body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: status %d", method, resp.StatusCode)
	}
	var out map[string]any
	decodeBody(t, resp, &out)
	return out
}

func TestMCPRequiresBearerAuth(t *testing.T) {
	env := newTestEnv(t)
	resp := doJSONWithHeaders(t, &http.Client{}, "POST", env.url("/mcp"), map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestMCPInitializeAndToolsList(t *testing.T) {
	env, token := mcpTokenEnv(t)

	initResult := mcpCall(t, env, token, 1, "initialize", map[string]any{"protocolVersion": "2025-06-18"})
	result, ok := initResult["result"].(map[string]any)
	if !ok || result["serverInfo"] == nil {
		t.Fatalf("initialize result = %+v", initResult)
	}

	listResult := mcpCall(t, env, token, 2, "tools/list", nil)
	res, ok := listResult["result"].(map[string]any)
	if !ok {
		t.Fatalf("tools/list result = %+v", listResult)
	}
	tools, ok := res["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("tools/list returned no tools: %+v", res)
	}

	foundCreateNetwork := false
	for _, tool := range tools {
		m, _ := tool.(map[string]any)
		if m["name"] == "create_network" {
			foundCreateNetwork = true
			if m["inputSchema"] == nil {
				t.Errorf("create_network tool has no inputSchema: %+v", m)
			}
		}
	}
	if !foundCreateNetwork {
		t.Errorf("create_network not advertised in tools/list: %+v", tools)
	}
}

func TestMCPUnknownMethodReturnsJSONRPCError(t *testing.T) {
	env, token := mcpTokenEnv(t)
	out := mcpCall(t, env, token, 1, "not/a/real/method", nil)
	errObj, ok := out["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected a JSON-RPC error, got %+v", out)
	}
	if errObj["code"] != float64(-32601) {
		t.Errorf("error code = %v, want -32601", errObj["code"])
	}
}

func TestMCPNotificationGetsNoBody(t *testing.T) {
	env, token := mcpTokenEnv(t)
	resp := doBearer(t, token, "POST", env.url("/mcp"), map[string]any{
		"jsonrpc": "2.0", "method": "notifications/initialized",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
}

func toolCallResultText(t *testing.T, out map[string]any) (string, bool) {
	t.Helper()
	result, ok := out["result"].(map[string]any)
	if !ok {
		t.Fatalf("tools/call did not return a result: %+v", out)
	}
	isErr, _ := result["isError"].(bool)
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("tools/call result has no content: %+v", result)
	}
	block, _ := content[0].(map[string]any)
	text, _ := block["text"].(string)
	return text, isErr
}

func TestMCPCreateAndListNetworks(t *testing.T) {
	env, token := mcpTokenEnv(t)

	createOut := mcpCall(t, env, token, 1, "tools/call", map[string]any{
		"name":      "create_network",
		"arguments": map[string]any{"name": "office"},
	})
	text, isErr := toolCallResultText(t, createOut)
	if isErr {
		t.Fatalf("create_network returned an error: %s", text)
	}
	var created map[string]any
	if err := json.Unmarshal([]byte(text), &created); err != nil {
		t.Fatalf("decoding create_network result: %v", err)
	}
	nwid, _ := created["nwid"].(string)
	if nwid == "" {
		t.Fatalf("created network has no nwid: %+v", created)
	}

	listOut := mcpCall(t, env, token, 2, "tools/call", map[string]any{"name": "list_networks"})
	listText, isErr := toolCallResultText(t, listOut)
	if isErr {
		t.Fatalf("list_networks returned an error: %s", listText)
	}
	var networks []map[string]any
	if err := json.Unmarshal([]byte(listText), &networks); err != nil {
		t.Fatalf("decoding list_networks result: %v", err)
	}
	if len(networks) != 1 || networks[0]["nwid"] != nwid {
		t.Fatalf("list_networks = %+v, want only %s", networks, nwid)
	}

	// The MCP mutation must have left an audit trail attributing the
	// action to the token's owner.
	entries, err := env.audit.Recent(1)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != 1 || entries[0].User != "alice" || entries[0].Action != "created network" {
		t.Fatalf("audit entries = %+v", entries)
	}
}

func TestMCPUnknownToolReturnsProtocolError(t *testing.T) {
	env, token := mcpTokenEnv(t)
	out := mcpCall(t, env, token, 1, "tools/call", map[string]any{"name": "not_a_real_tool"})
	errObj, ok := out["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected a JSON-RPC error, got %+v", out)
	}
	if errObj["code"] != float64(-32602) {
		t.Errorf("error code = %v, want -32602", errObj["code"])
	}
}

func TestMCPToolExecutionErrorIsNotAProtocolError(t *testing.T) {
	env, token := mcpTokenEnv(t)
	out := mcpCall(t, env, token, 1, "tools/call", map[string]any{
		"name":      "get_network",
		"arguments": map[string]any{"network_id": "doesnotexist"},
	})
	if out["error"] != nil {
		t.Fatalf("expected a successful JSON-RPC response with isError, got a protocol error: %+v", out)
	}
	text, isErr := toolCallResultText(t, out)
	if !isErr {
		t.Fatalf("expected isError: true for a nonexistent network, text=%q", text)
	}
}

func TestMCPOrganizationToolsRespectMembership(t *testing.T) {
	env, token := mcpTokenEnv(t)
	c := env.loginAs(t, "alice", "alice-password1")

	var org map[string]any
	resp := doJSON(t, c, "POST", env.url("/api/organizations"), map[string]any{"name": "Acme"})
	decodeBody(t, resp, &org)
	orgID, _ := org["id"].(string)

	out := mcpCall(t, env, token, 1, "tools/call", map[string]any{
		"name":      "get_organization",
		"arguments": map[string]any{"organization_id": orgID},
	})
	text, isErr := toolCallResultText(t, out)
	if isErr {
		t.Fatalf("get_organization for a member returned an error: %s", text)
	}

	// A second user, who never joined the org, is forbidden.
	outsiderC := env.loginAs(t, "mallory", "mallory-password1")
	outsiderTokenResp := doJSON(t, outsiderC, "POST", env.url("/api/tokens"), map[string]any{"name": "outsider"})
	var outsiderCreated map[string]any
	decodeBody(t, outsiderTokenResp, &outsiderCreated)
	outsiderToken, _ := outsiderCreated["token"].(string)

	forbiddenOut := mcpCall(t, env, outsiderToken, 1, "tools/call", map[string]any{
		"name":      "get_organization",
		"arguments": map[string]any{"organization_id": orgID},
	})
	_, isErr = toolCallResultText(t, forbiddenOut)
	if !isErr {
		t.Fatalf("expected isError: true for a non-member, got %+v", forbiddenOut)
	}
}
