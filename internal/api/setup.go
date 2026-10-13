package api

import "net/http"

// handleSetupStatus tells the frontend whether to show the first-run
// setup page instead of the login screen. It's intentionally
// unauthenticated -- there's no session to have yet before setup runs
// -- but only reports the ZeroTier controller's reachability while
// setup is still pending, so a configured instance doesn't leak that
// detail to an anonymous caller.
func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	count, err := s.users.Count()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not check setup status")
		return
	}
	if count > 0 {
		writeJSON(w, http.StatusOK, map[string]any{"needsSetup": false})
		return
	}

	resp := map[string]any{"needsSetup": true}
	status, statusErr := s.zt.Status()
	if statusErr != nil {
		resp["controllerReachable"] = false
		resp["controllerError"] = statusErr.Error()
	} else {
		resp["controllerReachable"] = true
		resp["controllerVersion"] = status["version"]
	}
	writeJSON(w, http.StatusOK, resp)
}

type setupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleSetup creates the first admin account and -- unlike every
// other account, which starts with mustChangePassword -- logs them
// straight in, since they just chose this password themselves.
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	count, err := s.users.Count()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not check setup status")
		return
	}
	if count > 0 {
		writeError(w, http.StatusConflict, "ztpanel is already configured")
		return
	}

	var req setupRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "a username and a password of at least 8 characters are required")
		return
	}

	if err := s.users.Create(req.Username, req.Password, false); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create the admin account")
		return
	}
	s.audit.Record(req.Username, "completed first-run setup", req.Username, "")

	s.issueSessionCookie(w, r, req.Username)
	writeJSON(w, http.StatusCreated, map[string]any{"username": req.Username})
}
