package handlers

import (
    "RTF/database"
    "RTF/models"
    "encoding/json"
    "net/http"
)

func ServeProfile(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodGet {
        userID := getUserIDFromSession(r)
        if userID == 0 {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

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
}
