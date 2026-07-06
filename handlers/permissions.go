package handlers

import (
	"RTF/database"
	"database/sql"
	"fmt"
	"net/http"
	"os"
)

type rowQuerier interface {
	QueryRow(string, ...interface{}) *sql.Row
}

func roleIn(db rowQuerier, id int) (string, error) {
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE id=?", id).Scan(&role)
	if err == nil && configuredAdmin(id) {
		role = "admin"
	}
	return role, err
}
func userRole(id int) string     { role, _ := roleIn(database.DBInstance.DB, id); return role }
func staffRole(role string) bool { return role == "admin" || role == "moderator" }
func viewer(r *http.Request) (int, bool) {
	s, err := lookupSession(requestToken(r))
	if err != nil {
		return 0, false
	}
	return s.UserID, staffRole(userRole(s.UserID))
}
func visibility(alias string, r *http.Request) string {
	id, staff := viewer(r)
	if staff {
		return "1=1"
	}
	return fmt.Sprintf("(%s.status='published' OR %s.user_id=%d)", alias, alias, id)
}
func submissionStatus(tx *sql.Tx, id int) (string, error) {
	role, err := roleIn(tx, id)
	if err != nil {
		return "", err
	}
	if os.Getenv("FORUM_PREMODERATE") == "true" && !staffRole(role) {
		return "pending", nil
	}
	return "published", nil
}
func contentChanged()   { broadcastMessage(Message{Type: "content_changed", Content: nil}) }
func communityChanged() { broadcastMessage(Message{Type: "community_changed", Content: nil}) }
