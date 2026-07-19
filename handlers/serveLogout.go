package handlers

import (
	"RTF/database"
	"net/http"
	"time"
)

func ServeLogout(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	if _, err := database.DBInstance.DB.Exec("DELETE FROM sessions WHERE session_token=?", s.Token); err != nil {
		serverError(w, err)
		return
	}
	closeSessionConnections(s.Token)
	http.SetCookie(w, &http.Cookie{Name: "session_token", Value: "", Path: "/", HttpOnly: true, Secure: secureCookie(r), SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	w.WriteHeader(http.StatusNoContent)
}
