package handlers

import (
	"RTF/database"
	"errors"
	"github.com/google/uuid"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,32}$`)
var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func ServeRegister(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}
	if r.Method == http.MethodGet {
		serveShell(w, r)
		return
	}
	var user struct {
		Username  string `json:"username"`
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Age       int    `json:"age"`
		Gender    string `json:"gender"`
	}
	if !decodeJSON(w, r, &user) {
		return
	}
	user.Username = strings.TrimSpace(user.Username)
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	user.FirstName = strings.TrimSpace(user.FirstName)
	user.LastName = strings.TrimSpace(user.LastName)
	if !usernamePattern.MatchString(user.Username) {
		http.Error(w, "Username must be 3–32 letters, numbers, dots, underscores, or hyphens", http.StatusBadRequest)
		return
	}
	if len(user.Email) > 254 || !emailPattern.MatchString(user.Email) {
		http.Error(w, "Enter a valid email address", http.StatusBadRequest)
		return
	}
	if user.FirstName == "" || user.LastName == "" || len([]rune(user.FirstName)) > 80 || len([]rune(user.LastName)) > 80 {
		http.Error(w, "First and last names must contain 1–80 characters", http.StatusBadRequest)
		return
	}
	if user.Age < 1 || user.Age > 120 || (user.Gender != "male" && user.Gender != "female") {
		http.Error(w, "Enter a valid age and gender", http.StatusBadRequest)
		return
	}
	if len([]rune(user.Password)) < 8 || len(user.Password) > 72 {
		http.Error(w, "Password must contain at least 8 characters and no more than 72 bytes", http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		serverError(w, err)
		return
	}
	tx, err := database.DBInstance.DB.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	result, err := tx.Exec("INSERT INTO users(username,email,password,first_name,last_name,age,gender) VALUES(?,?,?,?,?,?,?)", user.Username, user.Email, string(hash), user.FirstName, user.LastName, user.Age, user.Gender)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			http.Error(w, "Username or email already exists", http.StatusConflict)
		} else {
			serverError(w, err)
		}
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		serverError(w, err)
		return
	}
	token, expires := uuid.NewString(), time.Now().UTC().Add(24*time.Hour)
	if _, err = tx.Exec("INSERT INTO sessions(session_token,email,expires_at) VALUES(?,?,?)", token, user.Email, expires); err != nil {
		serverError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	setSessionCookie(w, r, token, expires)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"user_id": id, "username": user.Username, "is_admin": isAdmin(int(id))})
}
