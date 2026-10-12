package api

import (
	"net/http"

	"github.com/antonio-damaso/ztpanel/internal/orgsvc"
)

func (s *Server) handleListOrganizations(w http.ResponseWriter, r *http.Request) {
	orgs, err := s.orgs.ListForUser(userFromContext(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list organizations")
		return
	}
	writeJSON(w, http.StatusOK, orgs)
}

type createOrgRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateOrganization(w http.ResponseWriter, r *http.Request) {
	var req createOrgRequest
	if err := decodeJSON(r, &req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "organization name is required")
		return
	}
	org, err := s.orgs.Create(req.Name, userFromContext(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create organization")
		return
	}
	writeJSON(w, http.StatusCreated, org)
}

// requireOrgMember loads the organization in the URL and checks the
// current user belongs to it, attaching both to the request context.
func (s *Server) requireOrgMember(next func(w http.ResponseWriter, r *http.Request, org *orgsvc.Organization, role orgsvc.Role)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := r.PathValue("orgId")
		org, err := s.orgs.Get(orgID)
		if err != nil {
			writeError(w, http.StatusNotFound, "organization not found")
			return
		}
		username := userFromContext(r)
		role, err := s.orgs.MemberRole(orgID, username)
		if err != nil || role == "" {
			writeError(w, http.StatusForbidden, "not a member of this organization")
			return
		}
		next(w, r, org, role)
	}
}

func (s *Server) handleGetOrganization(w http.ResponseWriter, r *http.Request) {
	s.requireOrgMember(func(w http.ResponseWriter, r *http.Request, org *orgsvc.Organization, _ orgsvc.Role) {
		writeJSON(w, http.StatusOK, org)
	})(w, r)
}

func (s *Server) handleDeleteOrganization(w http.ResponseWriter, r *http.Request) {
	s.requireOrgMember(func(w http.ResponseWriter, r *http.Request, org *orgsvc.Organization, role orgsvc.Role) {
		if role != orgsvc.RoleAdmin {
			writeError(w, http.StatusForbidden, "only an organization admin can delete it")
			return
		}
		if err := s.orgs.Delete(org.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not delete organization")
			return
		}
		writeJSON(w, http.StatusOK, nil)
	})(w, r)
}

type addMemberRequest struct {
	Username string      `json:"username"`
	Role     orgsvc.Role `json:"role"`
}

func (s *Server) handleAddOrgMember(w http.ResponseWriter, r *http.Request) {
	s.requireOrgMember(func(w http.ResponseWriter, r *http.Request, org *orgsvc.Organization, role orgsvc.Role) {
		if role != orgsvc.RoleAdmin {
			writeError(w, http.StatusForbidden, "only an organization admin can add members")
			return
		}
		var req addMemberRequest
		if err := decodeJSON(r, &req); err != nil || req.Username == "" {
			writeError(w, http.StatusBadRequest, "username is required")
			return
		}
		if req.Role != orgsvc.RoleAdmin && req.Role != orgsvc.RoleUser {
			req.Role = orgsvc.RoleUser
		}
		if err := s.orgs.AddMember(org.ID, req.Username, req.Role); err != nil {
			writeError(w, http.StatusConflict, "could not add member: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, nil)
	})(w, r)
}

func (s *Server) handleRemoveOrgMember(w http.ResponseWriter, r *http.Request) {
	s.requireOrgMember(func(w http.ResponseWriter, r *http.Request, org *orgsvc.Organization, role orgsvc.Role) {
		if role != orgsvc.RoleAdmin {
			writeError(w, http.StatusForbidden, "only an organization admin can remove members")
			return
		}
		username := r.PathValue("username")
		if err := s.orgs.RemoveMember(org.ID, username); err != nil {
			writeError(w, http.StatusNotFound, "member not found")
			return
		}
		writeJSON(w, http.StatusOK, nil)
	})(w, r)
}

// handleListOrgNetworks returns only the networks assigned to this
// organization -- the same shape as GET /api/networks, filtered.
func (s *Server) handleListOrgNetworks(w http.ResponseWriter, r *http.Request) {
	s.requireOrgMember(func(w http.ResponseWriter, r *http.Request, org *orgsvc.Organization, _ orgsvc.Role) {
		belongs := map[string]bool{}
		for _, id := range org.NetworkIDs {
			belongs[id] = true
		}

		ids, err := s.zt.ListNetworkIDs()
		if err != nil {
			writeError(w, http.StatusBadGateway, "could not list networks: "+err.Error())
			return
		}

		networks := make([]map[string]any, 0)
		for _, id := range ids {
			if !belongs[id] {
				continue
			}
			net, err := s.zt.GetNetwork(id)
			if err != nil {
				continue
			}
			if members, err := s.zt.ListMemberIDs(id); err == nil {
				net["memberCount"] = len(members)
			}
			networks = append(networks, net)
		}
		writeJSON(w, http.StatusOK, networks)
	})(w, r)
}

// handleAssignNetworkToOrg moves a network (already created via
// POST /api/networks) into this organization.
func (s *Server) handleAssignNetworkToOrg(w http.ResponseWriter, r *http.Request) {
	s.requireOrgMember(func(w http.ResponseWriter, r *http.Request, org *orgsvc.Organization, role orgsvc.Role) {
		if role != orgsvc.RoleAdmin {
			writeError(w, http.StatusForbidden, "only an organization admin can assign networks")
			return
		}
		networkID := r.PathValue("nwid")
		if _, err := s.zt.GetNetwork(networkID); err != nil {
			writeError(w, http.StatusNotFound, "network not found")
			return
		}
		if err := s.orgs.AssignNetwork(org.ID, networkID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not assign network")
			return
		}
		writeJSON(w, http.StatusOK, nil)
	})(w, r)
}

func (s *Server) handleUnassignNetworkFromOrg(w http.ResponseWriter, r *http.Request) {
	s.requireOrgMember(func(w http.ResponseWriter, r *http.Request, org *orgsvc.Organization, role orgsvc.Role) {
		if role != orgsvc.RoleAdmin {
			writeError(w, http.StatusForbidden, "only an organization admin can unassign networks")
			return
		}
		networkID := r.PathValue("nwid")
		if err := s.orgs.UnassignNetwork(networkID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not unassign network")
			return
		}
		writeJSON(w, http.StatusOK, nil)
	})(w, r)
}
