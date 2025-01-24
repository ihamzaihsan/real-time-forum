package handlers

import (
	"RTF/database"
	"net/http"
	"time"
)

func ServeLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the session token from the request header
	sessionToken := r.Header.Get("Authorization")
	if sessionToken == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Delete the session from the database
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

	// Clear the session cookie by setting it to an expired value
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",  // Replace with your actual cookie name
		Value:    "",               // Set an empty value
		Path:     "/",              // Make sure it's valid for the entire domain
		HttpOnly: true,             // Ensure it's inaccessible to JavaScript
		Secure:   true,             // Use only over HTTPS
		SameSite: http.SameSiteStrictMode, // Adjust SameSite according to your needs
		Expires:  time.Unix(0, 0),  // Set the expiration to a past date
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Logged out successfully"))
}
