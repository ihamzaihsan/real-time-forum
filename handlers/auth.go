package handlers

import (
	"RTF/database"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type session struct {
	UserID                 int
	Username, Email, Token string
	Expires                time.Time
}

func requestToken(r *http.Request) string {
	if token := r.Header.Get("Authorization"); token != "" {
		return strings.TrimPrefix(token, "Bearer ")
	}
	if cookie, err := r.Cookie("session_token"); err == nil {
		return cookie.Value
	}
	return ""
}

func lookupSession(token string) (session, error) {
	var s session
	if token == "" {
		return s, sql.ErrNoRows
	}
	err := database.DBInstance.DB.QueryRow(`SELECT u.id, u.username, u.email, s.expires_at FROM users u JOIN sessions s ON u.email=s.email WHERE s.session_token=?`, token).Scan(&s.UserID, &s.Username, &s.Email, &s.Expires)
	if err != nil {
		return session{}, err
	}
	if !time.Now().Before(s.Expires) {
		return session{}, sql.ErrNoRows
	}
	s.Token = token
	return s, nil
}

func requireSession(w http.ResponseWriter, r *http.Request) (session, bool) {
	s, err := lookupSession(requestToken(r))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Sign in to continue", http.StatusUnauthorized)
		} else {
			serverError(w, err)
		}
		return session{}, false
	}
	return s, true
}

func secureCookie(r *http.Request) bool { return r.TLS != nil || os.Getenv("COOKIE_SECURE") == "true" }
func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{Name: "session_token", Value: token, Path: "/", HttpOnly: true, Secure: secureCookie(r), SameSite: http.SameSiteStrictMode, Expires: expires})
}

func createSession(email string) (string, time.Time, error) {
	token, expires := uuid.NewString(), time.Now().UTC().Add(24*time.Hour)
	_, err := database.DBInstance.DB.Exec("INSERT INTO sessions(session_token,email,expires_at) VALUES(?,?,?)", token, email, expires)
	return token, expires, err
}

func isAdmin(userID int) bool {
	for _, value := range strings.Split(os.Getenv("ADMIN_USER_IDS"), ",") {
		id, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil && id == userID && id > 0 {
			return true
		}
	}
	return false
}
