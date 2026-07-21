package handlers

import (
	"RTF/database"
	"database/sql"
	"errors"
	"github.com/mattn/go-sqlite3"
	"net/http"
	"net/mail"
	"strings"
)

var errEmailRegistered = errors.New("email already registered")

func validEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email && len(email) <= 254 && emailPattern.MatchString(email)
}

func resolveOAuth(provider, subject, email string) (string, error) {
	var existing string
	err := database.DBInstance.DB.QueryRow(`SELECT u.email FROM oauth_identities i JOIN users u ON u.id=i.user_id WHERE i.provider=? AND i.subject=?`, provider, subject).Scan(&existing)
	if !errors.Is(err, sql.ErrNoRows) {
		return existing, err
	}
	var count int
	if err := database.DBInstance.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email=? COLLATE NOCASE", email).Scan(&count); err != nil {
		return "", err
	}
	if count > 0 {
		return "", errEmailRegistered
	}
	return "", sql.ErrNoRows
}

func (o *OAuth) Providers(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	providers := []string{}
	if o.err == nil {
		for _, name := range []string{"google", "github"} {
			if _, ok := o.providers[name]; ok {
				providers = append(providers, name)
			}
		}
	}
	writeJSON(w, 200, providers)
}

func CompleteOAuthProfile(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}
	if r.Method == http.MethodGet {
		serveShell(w, r)
		return
	}
	cookie, err := r.Cookie("__Host-oauth-profile")
	if err != nil {
		http.Error(w, "Provider sign-in expired. Start again.", 401)
		return
	}
	var input struct {
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Age       int    `json:"age"`
		Gender    string `json:"gender"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	if !usernamePattern.MatchString(input.Username) || input.FirstName == "" || input.LastName == "" || len([]rune(input.FirstName)) > 80 || len([]rune(input.LastName)) > 80 || input.Age < 1 || input.Age > 120 || (input.Gender != "male" && input.Gender != "female") {
		http.Error(w, "Complete the same required profile fields as local registration", 400)
		return
	}
	tx, err := database.DBInstance.DB.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	var provider, subject, email string
	err = tx.QueryRow("SELECT provider,subject,email FROM oauth_pending WHERE id=? AND julianday(expires_at)>julianday('now')", cookie.Value).Scan(&provider, &subject, &email)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Provider sign-in expired. Start again.", 401)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	var existing int
	if err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE email=? COLLATE NOCASE OR username=?", email, input.Username).Scan(&existing); err != nil {
		serverError(w, err)
		return
	}
	if existing > 0 {
		http.Error(w, "Username or email already exists; use the original sign-in method or choose another username", 409)
		return
	}
	result, err := tx.Exec("INSERT INTO users(username,email,password,first_name,last_name,age,gender) VALUES(?,?,'',?,?,?,?)", input.Username, email, input.FirstName, input.LastName, input.Age, input.Gender)
	if err != nil {
		var e sqlite3.Error
		if errors.As(err, &e) && e.Code == sqlite3.ErrConstraint {
			http.Error(w, "Account already exists. Restart sign-in.", 409)
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
	if _, err = tx.Exec("INSERT INTO oauth_identities(provider,subject,user_id) VALUES(?,?,?)", provider, subject, id); err != nil {
		serverError(w, err)
		return
	}
	if _, err = tx.Exec("DELETE FROM oauth_pending WHERE id=?", cookie.Value); err != nil {
		serverError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	token, expires, err := createSession(email)
	if err != nil {
		serverError(w, err)
		return
	}
	setSessionCookie(w, r, token, expires)
	http.SetCookie(w, &http.Cookie{Name: "__Host-oauth-profile", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	writeJSON(w, 201, map[string]interface{}{"user_id": id, "username": input.Username, "is_admin": false})
}
