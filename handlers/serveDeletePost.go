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
	tx, err := database.DBInstance.DB.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	var owner int
	var imageName, status string
	err = tx.QueryRow("SELECT user_id,COALESCE(image_path,''),status FROM posts WHERE id=?", id).Scan(&owner, &imageName, &status)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	role, err := roleIn(tx, s.UserID)
	if err != nil {
		serverError(w, err)
		return
	}
	if owner != s.UserID && !staffRole(role) {
		http.Error(w, "Only the author can delete this post", http.StatusForbidden)
		return
	}
	result, err := tx.Exec("DELETE FROM posts WHERE id=?", id)
	if err != nil {
		serverError(w, err)
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	if owner != s.UserID {
		if _, err := tx.Exec("INSERT INTO moderation_actions(moderator_id,post_id,action,reason) VALUES(?,?,'delete_post','Staff deletion')", s.UserID, id); err != nil {
			serverError(w, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	cleanupImage(imageName)
	if status == "published" {
		broadcastMessage(Message{Type: "post_deleted", Content: map[string]interface{}{"post_id": id}})
	} else {
		contentChanged()
	}
	communityChanged()
	writeJSON(w, http.StatusOK, map[string]interface{}{"post_id": id})
}
