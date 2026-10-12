package api

import "net/http"

func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := s.tokens.List(userFromContext(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list tokens")
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

type createTokenRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	var req createTokenRequest
	if err := decodeJSON(r, &req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "a name for the token is required")
		return
	}
	username := userFromContext(r)
	plaintext, token, err := s.tokens.Create(username, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create token")
		return
	}
	s.audit.Record(username, "created API token", req.Name, "")
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":        token.ID,
		"name":      token.Name,
		"createdAt": token.CreatedAt,
		// token is only ever returned here, at creation time.
		"token": plaintext,
	})
}

func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	username := userFromContext(r)
	id := r.PathValue("id")
	if err := s.tokens.Revoke(id, username); err != nil {
		writeError(w, http.StatusNotFound, "token not found")
		return
	}
	s.audit.Record(username, "revoked API token", id, "")
	writeJSON(w, http.StatusOK, nil)
}
