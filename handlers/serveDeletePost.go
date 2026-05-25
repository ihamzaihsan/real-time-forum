package handlers

import (
	"RTF/database"
	"database/sql"
	"errors"
	"net/http"
)

func ServeDeletePost(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	if !parseForm(w, r) {
		return
	}
	id, err := positiveID(r.FormValue("post_id"))
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}
	var owner int
	err = database.DBInstance.DB.QueryRow("SELECT user_id FROM posts WHERE id=?", id).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if owner != s.UserID {
		http.Error(w, "Only the author can delete this post", http.StatusForbidden)
		return
	}
	result, err := database.DBInstance.DB.Exec("DELETE FROM posts WHERE id=? AND user_id=?", id, s.UserID)
	if err != nil {
		serverError(w, err)
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	broadcastMessage(Message{Type: "post_deleted", Content: map[string]interface{}{"post_id": id}})
	writeJSON(w, http.StatusOK, map[string]interface{}{"post_id": id})
}
