package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"time"
)

func CheckAuth(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	s, err := lookupSession(requestToken(r))
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			serverError(w, err)
			return
		}
		setSessionCookie(w, r, "", time.Unix(1, 0))
		http.Error(w, "Sign in to continue", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"user_id": s.UserID, "username": s.Username, "is_admin": isAdmin(s.UserID)})
}
