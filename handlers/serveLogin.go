package handlers

import (
	"RTF/database"
	"database/sql"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
)

func ServeLogin(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}
	if r.Method == http.MethodGet {
		serveShell(w, r)
		return
	}
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	if input.Username == "" || len(input.Username) > 254 || input.Password == "" || len(input.Password) > 72 {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}
	var id int
	var username, email, password string
	err := database.DBInstance.DB.QueryRow("SELECT id,username,email,password FROM users WHERE username=? OR email=?", input.Username, input.Username).Scan(&id, &username, &email, &password)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(password), []byte(input.Password)) != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	token, expires, err := createSession(email)
	if err != nil {
		serverError(w, err)
		return
	}
	setSessionCookie(w, r, token, expires)
	writeJSON(w, http.StatusOK, map[string]interface{}{"user_id": id, "username": username, "is_admin": isAdmin(id), "role": userRole(id)})
}
