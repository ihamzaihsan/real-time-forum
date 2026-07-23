package handlers

import (
	"RTF/database"
	"RTF/models"
	"net/http"
	"strings"
)

func ServeProfile(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if r.URL.Path == "/profile" {
		serveShell(w, r)
		return
	}
	id, err := positiveID(strings.TrimPrefix(r.URL.Path, "/profile/"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	w.Header().Set("Vary", "Accept")
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		serveShell(w, r)
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	if s.UserID != id {
		http.Error(w, "You can only view your own profile", http.StatusForbidden)
		return
	}
	var profile models.UserProfile
	err = database.DBInstance.DB.QueryRow("SELECT first_name,last_name,username,email,age,gender FROM users WHERE id=?", id).Scan(&profile.FirstName, &profile.LastName, &profile.Username, &profile.Email, &profile.Age, &profile.Gender)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}
