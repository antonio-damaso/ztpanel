// MCP support: a single POST /mcp endpoint implementing the Model
// Context Protocol's Streamable HTTP transport (JSON-RPC 2.0, one
// request per response, no server-to-client streaming since every
// tool here is a quick, synchronous call). It is stateless -- no
// Mcp-Session-Id is issued -- and authenticated the same way as the
// rest of the API, via requireAuth (in practice, always an
// Authorization: Bearer <api token> header here, since MCP clients
// don't carry browser cookies).
//
// Tool scope is deliberately asymmetric: full CRUD on networks and
// members (the actual point of letting an LLM drive ztpanel), but
// organizations/activity are read-only here. Creating/deleting
// organizations and changing membership stays a web-UI-only action,
// so an LLM can't restructure who has access to what.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/antonio-damaso/ztpanel/internal/orgsvc"
)

const mcpProtocolVersion = "2025-06-18"

type jsonrpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	var msg jsonrpcMessage
	if err := decodeJSON(r, &msg); err != nil {
		writeJSONRPCError(w, http.StatusBadRequest, nil, -32700, "parse error")
		return
	}

	isNotification := len(msg.ID) == 0 || string(msg.ID) == "null"
	username := userFromContext(r)

	result, rpcErr := s.dispatchMCP(msg.Method, msg.Params, username)

	if isNotification {
		// JSON-RPC notifications get no response body at all.
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if rpcErr != nil {
		writeJSONRPCError(w, http.StatusOK, msg.ID, rpcErr.Code, rpcErr.Message)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"jsonrpc": "2.0",
		"id":      json.RawMessage(msg.ID),
		"result":  result,
	})
}

func writeJSONRPCError(w http.ResponseWriter, status int, id json.RawMessage, code int, message string) {
	body := map[string]any{"jsonrpc": "2.0", "error": jsonrpcError{Code: code, Message: message}}
	if id != nil {
		body["id"] = id
	} else {
		body["id"] = nil
	}
	writeJSON(w, status, body)
}

func (s *Server) dispatchMCP(method string, params json.RawMessage, username string) (any, *jsonrpcError) {
	switch method {
	case "initialize":
		return map[string]any{
			"protocolVersion": mcpProtocolVersion,
			"serverInfo":      map[string]any{"name": "ztpanel", "version": "0.1.0"},
			"capabilities":    map[string]any{"tools": map[string]any{}},
		}, nil

	case "notifications/initialized":
		return nil, nil

	case "ping":
		return map[string]any{}, nil

	case "tools/list":
		return map[string]any{"tools": mcpTools}, nil

	case "tools/call":
		return s.handleToolCall(params, username)

	default:
		return nil, &jsonrpcError{Code: -32601, Message: "method not found: " + method}
	}
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func (s *Server) handleToolCall(params json.RawMessage, username string) (any, *jsonrpcError) {
	var p toolCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &jsonrpcError{Code: -32602, Message: "invalid params"}
	}

	tool, ok := mcpToolHandlers[p.Name]
	if !ok {
		return nil, &jsonrpcError{Code: -32602, Message: "unknown tool: " + p.Name}
	}
	if p.Arguments == nil {
		p.Arguments = map[string]any{}
	}

	result, err := tool(s, username, p.Arguments)
	if err != nil {
		// A tool-execution failure is a normal (if unsuccessful) MCP
		// result, not a JSON-RPC protocol error -- it lets the model
		// see what went wrong and decide what to do next.
		return map[string]any{
			"content": []map[string]any{{"type": "text", "text": err.Error()}},
			"isError": true,
		}, nil
	}

	text, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return nil, &jsonrpcError{Code: -32603, Message: "internal error"}
	}
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": string(text)}},
	}, nil
}

// ---- argument helpers ----

func argString(args map[string]any, key string) string {
	s, _ := args[key].(string)
	return s
}

func argObject(args map[string]any, key string) map[string]any {
	if m, ok := args[key].(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func argLimit(args map[string]any, key string, def int) int {
	n, ok := args[key].(float64) // JSON numbers decode as float64
	if !ok || n <= 0 {
		return def
	}
	return int(n)
}

// ---- tool catalog ----

type mcpToolHandler func(s *Server, username string, args map[string]any) (any, error)

var mcpToolHandlers = map[string]mcpToolHandler{
	"list_networks":              func(s *Server, _ string, _ map[string]any) (any, error) { return s.personalNetworks() },
	"get_network":                func(s *Server, _ string, a map[string]any) (any, error) { return s.zt.GetNetwork(argString(a, "network_id")) },
	"create_network":             mcpCreateNetwork,
	"update_network":             mcpUpdateNetwork,
	"delete_network":             mcpDeleteNetwork,
	"list_members":               func(s *Server, _ string, a map[string]any) (any, error) { return s.networkMembers(argString(a, "network_id")) },
	"get_member":                 func(s *Server, _ string, a map[string]any) (any, error) { return s.zt.GetMember(argString(a, "network_id"), argString(a, "address")) },
	"update_member":              mcpUpdateMember,
	"delete_member":              mcpDeleteMember,
	"list_organizations":         func(s *Server, username string, _ map[string]any) (any, error) { return s.orgs.ListForUser(username) },
	"get_organization":           mcpGetOrganization,
	"list_organization_networks": mcpListOrganizationNetworks,
	"get_activity":               func(s *Server, _ string, a map[string]any) (any, error) { return s.audit.Recent(argLimit(a, "limit", 50)) },
}

func mcpCreateNetwork(s *Server, username string, args map[string]any) (any, error) {
	net, err := s.zt.UpsertNetwork("______", map[string]any{"name": argString(args, "name")})
	if err != nil {
		return nil, err
	}
	s.audit.Record(username, "created network", toString(net["name"])+" ("+toString(net["nwid"])+")", "via MCP")
	return net, nil
}

func mcpUpdateNetwork(s *Server, username string, args map[string]any) (any, error) {
	id := argString(args, "network_id")
	fields := argObject(args, "fields")
	net, err := s.zt.UpsertNetwork(id, fields)
	if err != nil {
		return nil, err
	}
	s.audit.Record(username, "updated network", toString(net["name"])+" ("+id+")", changedFieldNames(fields)+" (via MCP)")
	return net, nil
}

func mcpDeleteNetwork(s *Server, username string, args map[string]any) (any, error) {
	id := argString(args, "network_id")
	name := id
	if net, err := s.zt.GetNetwork(id); err == nil {
		name = toString(net["name"]) + " (" + id + ")"
	}
	if err := s.zt.DeleteNetwork(id); err != nil {
		return nil, err
	}
	s.audit.Record(username, "deleted network", name, "via MCP")
	return map[string]any{"deleted": true}, nil
}

func mcpUpdateMember(s *Server, username string, args map[string]any) (any, error) {
	networkID, addr := argString(args, "network_id"), argString(args, "address")
	fields := argObject(args, "fields")
	member, err := s.zt.UpsertMember(networkID, addr, fields)
	if err != nil {
		return nil, err
	}
	s.audit.Record(username, "updated member", addr+" on "+networkID, changedFieldNames(fields)+" (via MCP)")
	return member, nil
}

func mcpDeleteMember(s *Server, username string, args map[string]any) (any, error) {
	networkID, addr := argString(args, "network_id"), argString(args, "address")
	if err := s.zt.DeleteMember(networkID, addr); err != nil {
		return nil, err
	}
	s.audit.Record(username, "removed member", addr+" from "+networkID, "via MCP")
	return map[string]any{"deleted": true}, nil
}

func mcpGetOrganization(s *Server, username string, args map[string]any) (any, error) {
	org, _, err := s.orgMembership(argString(args, "organization_id"), username)
	if err != nil {
		return nil, mcpOrgError(err)
	}
	return org, nil
}

func mcpListOrganizationNetworks(s *Server, username string, args map[string]any) (any, error) {
	org, _, err := s.orgMembership(argString(args, "organization_id"), username)
	if err != nil {
		return nil, mcpOrgError(err)
	}
	return s.organizationNetworks(org)
}

func mcpOrgError(err error) error {
	if err == orgsvc.ErrNotFound {
		return errors.New("organization not found")
	}
	return errors.New("not a member of this organization")
}

// mcpTools describes every tool above for tools/list, as plain JSON
// Schema. Kept as data right next to the handler map above so a new
// tool is always added in both places at once.
var mcpTools = []map[string]any{
	{
		"name":        "list_networks",
		"description": "List ZeroTier networks not assigned to any organization (personal/ungrouped networks).",
		"inputSchema": objSchema(nil, nil),
	},
	{
		"name":        "get_network",
		"description": "Get full detail for one ZeroTier network by its 16-character network ID.",
		"inputSchema": objSchema(map[string]any{"network_id": strSchema("The network's 16-character ID")}, []string{"network_id"}),
	},
	{
		"name":        "create_network",
		"description": "Create a new ZeroTier network.",
		"inputSchema": objSchema(map[string]any{"name": strSchema("A human-readable name for the network")}, []string{"name"}),
	},
	{
		"name":        "update_network",
		"description": "Update a ZeroTier network. `fields` is merged into the network's config by the controller (e.g. name, private, routes, ipAssignmentPools, v4AssignMode, dns, authTokens).",
		"inputSchema": objSchema(map[string]any{
			"network_id": strSchema("The network's 16-character ID"),
			"fields":     map[string]any{"type": "object", "description": "Partial network object to merge, e.g. {\"name\": \"new-name\", \"private\": true}"},
		}, []string{"network_id", "fields"}),
	},
	{
		"name":        "delete_network",
		"description": "Permanently delete a ZeroTier network.",
		"inputSchema": objSchema(map[string]any{"network_id": strSchema("The network's 16-character ID")}, []string{"network_id"}),
	},
	{
		"name":        "list_members",
		"description": "List every member of a network, enriched with live online/offline/latency status.",
		"inputSchema": objSchema(map[string]any{"network_id": strSchema("The network's 16-character ID")}, []string{"network_id"}),
	},
	{
		"name":        "get_member",
		"description": "Get full detail for one network member by its 10-character ZeroTier address.",
		"inputSchema": objSchema(map[string]any{
			"network_id": strSchema("The network's 16-character ID"),
			"address":    strSchema("The member's 10-character ZeroTier address"),
		}, []string{"network_id", "address"}),
	},
	{
		"name":        "update_member",
		"description": "Update a network member. `fields` is merged into the member's config (e.g. authorized, activeBridge, name, ipAssignments).",
		"inputSchema": objSchema(map[string]any{
			"network_id": strSchema("The network's 16-character ID"),
			"address":    strSchema("The member's 10-character ZeroTier address"),
			"fields":     map[string]any{"type": "object", "description": "Partial member object to merge, e.g. {\"authorized\": true}"},
		}, []string{"network_id", "address", "fields"}),
	},
	{
		"name":        "delete_member",
		"description": "Remove a member from a network.",
		"inputSchema": objSchema(map[string]any{
			"network_id": strSchema("The network's 16-character ID"),
			"address":    strSchema("The member's 10-character ZeroTier address"),
		}, []string{"network_id", "address"}),
	},
	{
		"name":        "list_organizations",
		"description": "List the organizations the authenticated user belongs to.",
		"inputSchema": objSchema(nil, nil),
	},
	{
		"name":        "get_organization",
		"description": "Get one organization's detail (members and assigned networks), if the authenticated user belongs to it.",
		"inputSchema": objSchema(map[string]any{"organization_id": strSchema("The organization's ID")}, []string{"organization_id"}),
	},
	{
		"name":        "list_organization_networks",
		"description": "List the networks assigned to one organization, if the authenticated user belongs to it.",
		"inputSchema": objSchema(map[string]any{"organization_id": strSchema("The organization's ID")}, []string{"organization_id"}),
	},
	{
		"name":        "get_activity",
		"description": "Get the most recent activity log entries (who did what, and when).",
		"inputSchema": objSchema(map[string]any{"limit": map[string]any{"type": "integer", "description": "Max entries to return (default 50)"}}, nil),
	},
}

func strSchema(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func objSchema(properties map[string]any, required []string) map[string]any {
	schema := map[string]any{"type": "object"}
	if properties != nil {
		schema["properties"] = properties
	} else {
		schema["properties"] = map[string]any{}
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
