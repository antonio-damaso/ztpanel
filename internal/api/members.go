package api

import (
	"net/http"
	"sort"
)

// handleListMembers returns every member of a network, each enriched
// with its live peer status (online/relay/offline, latency) so the
// UI can render connection state without a second round trip per row.
func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	networkID := r.PathValue("id")

	memberIDs, err := s.zt.ListMemberIDs(networkID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not list members: "+err.Error())
		return
	}

	peersByAddr := s.peersByAddress()

	members := make([]map[string]any, 0, len(memberIDs))
	for addr := range memberIDs {
		member, err := s.zt.GetMember(networkID, addr)
		if err != nil {
			continue
		}
		if peer, ok := peersByAddr[addr]; ok {
			member["peer"] = peer
		}
		members = append(members, member)
	}

	sort.Slice(members, func(i, j int) bool {
		return toString(members[i]["address"]) < toString(members[j]["address"])
	})

	writeJSON(w, http.StatusOK, members)
}

func (s *Server) peersByAddress() map[string]map[string]any {
	peers, err := s.zt.Peers()
	result := map[string]map[string]any{}
	if err != nil {
		return result
	}
	for _, p := range peers {
		if addr, ok := p["address"].(string); ok {
			result[addr] = p
		}
	}
	return result
}

func (s *Server) handleGetMember(w http.ResponseWriter, r *http.Request) {
	networkID, addr := r.PathValue("id"), r.PathValue("addr")
	member, err := s.zt.GetMember(networkID, addr)
	if err != nil {
		writeError(w, http.StatusNotFound, "member not found")
		return
	}
	writeJSON(w, http.StatusOK, member)
}

// handleUpdateMember forwards a partial member object to the
// controller (authorized, activeBridge, name, ipAssignments, ...).
func (s *Server) handleUpdateMember(w http.ResponseWriter, r *http.Request) {
	networkID, addr := r.PathValue("id"), r.PathValue("addr")
	var fields map[string]any
	if err := decodeJSON(r, &fields); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	member, err := s.zt.UpsertMember(networkID, addr, fields)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not update member: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, member)
}

func (s *Server) handleDeleteMember(w http.ResponseWriter, r *http.Request) {
	networkID, addr := r.PathValue("id"), r.PathValue("addr")
	if err := s.zt.DeleteMember(networkID, addr); err != nil {
		writeError(w, http.StatusBadGateway, "could not delete member: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}
