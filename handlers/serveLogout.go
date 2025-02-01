package handlers

import (
	"RTF/database"
	"net/http"
	"time"
)

func ServeLogout(w http.ResponseWriter, r *http.Request) {
	// For both GET and POST requests, redirect to login if no session token
	sessionToken := r.Header.Get("Authorization")
	if sessionToken == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Handle the actual logout logic
	_, err := database.DBInstance.DB.Exec("UPDATE users SET is_online = FALSE WHERE email IN (SELECT email FROM sessions WHERE session_token = ?)", sessionToken)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Delete the session from sessions table
	_, err = database.DBInstance.DB.Exec("DELETE FROM sessions WHERE session_token = ?", sessionToken)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
	})

	// Redirect to login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}