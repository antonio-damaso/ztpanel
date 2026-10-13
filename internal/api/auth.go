package api

import (
	"net/http"
	"time"

	"github.com/antonio-damaso/ztpanel/internal/authsvc"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := s.users.Authenticate(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	s.issueSessionCookie(w, user.Name)
	writeJSON(w, http.StatusOK, map[string]any{
		"username":           user.Name,
		"mustChangePassword": user.MustChangePassword,
	})
}

// issueSessionCookie is shared by handleLogin and handleSetup -- both
// end with the caller holding a valid session.
func (s *Server) issueSessionCookie(w http.ResponseWriter, username string) {
	token, expires := s.sessions.Issue(username)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})
	writeJSON(w, http.StatusOK, nil)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"username": userFromContext(r)})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (s *Server) handleChangeOwnPassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	username := userFromContext(r)

	if _, err := s.users.Authenticate(username, req.CurrentPassword); err != nil {
		writeError(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "new password must be at least 8 characters")
		return
	}
	if err := s.users.SetPassword(username, req.NewPassword, false); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update password")
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.users.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list users")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

type createUserRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "name is required and password must be at least 8 characters")
		return
	}
	err := s.users.Create(req.Name, req.Password, true)
	if err == authsvc.ErrUserExists {
		writeError(w, http.StatusConflict, "a user with that name already exists")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}
	s.audit.Record(userFromContext(r), "created panel user", req.Name, "")
	writeJSON(w, http.StatusCreated, nil)
}

type setPasswordRequest struct {
	Password           string `json:"password"`
	MustChangePassword bool   `json:"mustChangePassword"`
}

func (s *Server) handleSetUserPassword(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req setPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if err := s.users.SetPassword(name, req.Password, req.MustChangePassword); err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	s.audit.Record(userFromContext(r), "reset password", name, "")
	writeJSON(w, http.StatusOK, nil)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == userFromContext(r) {
		writeError(w, http.StatusBadRequest, "you may not delete your own account")
		return
	}
	if err := s.users.Delete(name); err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	s.audit.Record(userFromContext(r), "deleted panel user", name, "")
	writeJSON(w, http.StatusOK, nil)
}
