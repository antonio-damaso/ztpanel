package api

import (
	"net/http"
	"strconv"
)

func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	entries, err := s.audit.Recent(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load activity log")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}
