package api

import "net/http"

// handleHealthz is a plain liveness check for container orchestration
// and reverse proxies: no auth, no dependency on the ZeroTier
// controller, just "the process is up and answering HTTP requests".
// Controller reachability has its own authenticated check at
// GET /api/status, since it needs a logged-in user to be useful.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
