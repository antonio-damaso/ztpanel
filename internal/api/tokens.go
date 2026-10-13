package api

import (
	"net/http"
	"time"

	"github.com/antonio-damaso/ztpanel/internal/apitoken"
)

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
	// ReadOnly, if true, restricts the token to list/get-style calls
	// on both the REST API and MCP tools -- no creates, updates or
	// deletes.
	ReadOnly bool `json:"readOnly"`
	// ExpiresInDays, if positive, makes the token stop working after
	// that many days. Omitted or zero means it never expires.
	ExpiresInDays int `json:"expiresInDays"`
}

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	var req createTokenRequest
	if err := decodeJSON(r, &req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "a name for the token is required")
		return
	}
	if req.ExpiresInDays < 0 {
		writeError(w, http.StatusBadRequest, "expiresInDays cannot be negative")
		return
	}

	opts := apitoken.CreateOptions{ReadOnly: req.ReadOnly}
	if req.ExpiresInDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, req.ExpiresInDays)
		opts.ExpiresAt = &expiresAt
	}

	username := userFromContext(r)
	plaintext, token, err := s.tokens.Create(username, req.Name, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create token")
		return
	}
	s.audit.Record(username, "created API token", req.Name, "")
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":        token.ID,
		"name":      token.Name,
		"readOnly":  token.ReadOnly,
		"expiresAt": token.ExpiresAt,
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
