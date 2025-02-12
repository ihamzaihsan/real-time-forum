package handlers

import (
	database "RTF/database"
	"log"
	"net/http"
	"time"
)

func CheckAuth(w http.ResponseWriter, r *http.Request) {
	CleanExpiredSessions()
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var expiresAt time.Time
	err = database.DBInstance.DB.QueryRow(
		"SELECT expires_at FROM sessions WHERE session_token = ?",
		cookie.Value,
	).Scan(&expiresAt)

	if err != nil || time.Now().After(expiresAt) {
		// Clean up expired session
		database.DBInstance.DB.Exec("DELETE FROM sessions WHERE session_token = ?", cookie.Value)
		http.Error(w, "Session expired", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func CleanExpiredSessions() {
	_, err := database.DBInstance.DB.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
	if err != nil {
		log.Printf("Error cleaning expired sessions: %v", err)
	}
}
