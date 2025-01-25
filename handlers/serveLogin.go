package handlers

import (
	"RTF/database"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func ServeLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		http.ServeFile(w, r, filepath.Join("frontend", "index.html"))
		return
	}

	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	// Decode and validate input
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	loginReq.Username = strings.TrimSpace(loginReq.Username)
	loginReq.Password = strings.TrimSpace(loginReq.Password)

	if loginReq.Username == "" || loginReq.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Query user from database
	var user struct {
		ID       int
		Password string
		Email    string
	}
	err := database.DBInstance.DB.QueryRow(
		"SELECT id, password, email FROM users WHERE username = ? OR email = ?",
		loginReq.Username, loginReq.Username,
	).Scan(&user.ID, &user.Password, &user.Email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Verify password
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password)) != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate session token
	sessionToken := uuid.New().String()

	// Store session in database
	_, err = database.DBInstance.DB.Exec(`
		INSERT INTO sessions (session_token, email, expires_at) 
		VALUES (?, ?, ?)`,
		sessionToken, user.Email, time.Now().Add(24*time.Hour),
	)
	if err != nil {
		http.Error(w, "Session creation failed", http.StatusInternalServerError)
		return
	}

	// Update user's online status
	_, err = database.DBInstance.DB.Exec(`
		UPDATE users 
		SET is_online = TRUE, last_seen = CURRENT_TIMESTAMP 
		WHERE id = ?`, user.ID,
	)
	if err != nil {
		log.Printf("Failed to update online status: %v", err)
	}

	// Set secure cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set `true` in production with HTTPS
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	// Send success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful",
	})
}
