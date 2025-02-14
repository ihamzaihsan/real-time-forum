package handlers

import (
	"RTF/database"
	"RTF/models"
	"encoding/json"
	"net/http"
	"strings"
)

func ServeProfile(w http.ResponseWriter, r *http.Request) {
	// Check if this is a browser page request
	acceptHeader := r.Header.Get("Accept")
	isBrowserRequest := strings.Contains(acceptHeader, "text/html")

	if isBrowserRequest {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Continue with API response for profile data
	userID := strings.TrimPrefix(r.URL.Path, "/profile/")
	var profile models.UserProfile
	err := database.DBInstance.DB.QueryRow(
		`SELECT first_name, last_name, username, email, age, gender 
		FROM users WHERE id = ?`, userID).Scan(
		&profile.FirstName, &profile.LastName, &profile.Username,
		&profile.Email, &profile.Age, &profile.Gender)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}
