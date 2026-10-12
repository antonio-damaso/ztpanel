package api

import (
	"net/http"
	"sort"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.zt.Status()
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not reach the zerotier controller: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// handleListNetworks returns every network's full detail, not just
// IDs -- the UI's network list needs name/member-count/private at a
// glance, and the controller has no bulk "list with detail" endpoint,
// so ztpanel fetches each one. Controllers in this project's use case
// manage a handful of networks, not thousands, so this is cheap.
//
// Only "personal" networks are returned here -- ones not assigned to
// any organization. A network inside an organization is reached via
// GET /api/organizations/{orgId}/networks instead, so it's never
// listed twice.
func (s *Server) handleListNetworks(w http.ResponseWriter, r *http.Request) {
	networks, err := s.personalNetworks()
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not list networks: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, networks)
}

// personalNetworks is shared by handleListNetworks and the MCP
// list_networks tool.
func (s *Server) personalNetworks() ([]map[string]any, error) {
	ids, err := s.zt.ListNetworkIDs()
	if err != nil {
		return nil, err
	}
	assigned, err := s.orgs.AssignedNetworkIDs()
	if err != nil {
		return nil, err
	}

	networks := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		if assigned[id] {
			continue
		}
		net, err := s.zt.GetNetwork(id)
		if err != nil {
			continue
		}
		members, err := s.zt.ListMemberIDs(id)
		if err == nil {
			net["memberCount"] = len(members)
		}
		networks = append(networks, net)
	}

	sort.Slice(networks, func(i, j int) bool {
		return toString(networks[i]["name"]) < toString(networks[j]["name"])
	})
	return networks, nil
}

type createNetworkRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateNetwork(w http.ResponseWriter, r *http.Request) {
	var req createNetworkRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	net, err := s.zt.UpsertNetwork("______", map[string]any{"name": req.Name})
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not create network: "+err.Error())
		return
	}
	s.audit.Record(userFromContext(r), "created network", toString(net["name"])+" ("+toString(net["nwid"])+")", "")
	writeJSON(w, http.StatusCreated, net)
}

func (s *Server) handleGetNetwork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	net, err := s.zt.GetNetwork(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "network not found")
		return
	}
	writeJSON(w, http.StatusOK, net)
}

// handleUpdateNetwork accepts a partial network object and forwards
// it to the controller as-is. The controller itself validates and
// merges fields (name, private, routes, ipAssignmentPools,
// v4AssignMode, v6AssignMode, dns, authTokens, multicastLimit, ...),
// so ztpanel does not need to special-case each one.
func (s *Server) handleUpdateNetwork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var fields map[string]any
	if err := decodeJSON(r, &fields); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	net, err := s.zt.UpsertNetwork(id, fields)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not update network: "+err.Error())
		return
	}
	s.audit.Record(userFromContext(r), "updated network", toString(net["name"])+" ("+id+")", changedFieldNames(fields))
	writeJSON(w, http.StatusOK, net)
}

func (s *Server) handleDeleteNetwork(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	name := id
	if net, err := s.zt.GetNetwork(id); err == nil {
		name = toString(net["name"]) + " (" + id + ")"
	}
	if err := s.zt.DeleteNetwork(id); err != nil {
		writeError(w, http.StatusBadGateway, "could not delete network: "+err.Error())
		return
	}
	s.audit.Record(userFromContext(r), "deleted network", name, "")
	writeJSON(w, http.StatusOK, nil)
}

func changedFieldNames(fields map[string]any) string {
	names := make([]string, 0, len(fields))
	for k := range fields {
		names = append(names, k)
	}
	sort.Strings(names)
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}

func toString(v any) string {
	s, _ := v.(string)
	return s
}
