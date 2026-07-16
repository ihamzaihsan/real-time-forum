package handlers

import (
	"RTF/database"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
)

func ServeLike(w http.ResponseWriter, r *http.Request) { serveReaction(w, r, "post") }
func serveReaction(w http.ResponseWriter, r *http.Request, kind string) {
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
	column, table := kind+"_id", kind+"s"
	id, err := positiveID(r.FormValue(column))
	if err != nil {
		http.Error(w, "Invalid target ID", http.StatusBadRequest)
		return
	}
	like, err := strconv.ParseBool(r.FormValue("is_like"))
	if err != nil {
		http.Error(w, "Invalid reaction", http.StatusBadRequest)
		return
	}
	tx, err := database.DBInstance.DB.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	var target int
	query := "SELECT id FROM " + table + " WHERE id=? AND status='published'"
	if kind == "comment" {
		query = "SELECT c.id FROM comments c JOIN posts p ON p.id=c.post_id WHERE c.id=? AND c.status='published' AND p.status='published'"
	}
	err = tx.QueryRow(query, id).Scan(&target)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	_, err = tx.Exec("INSERT INTO likes(user_id,"+column+",is_like) VALUES(?,?,?) ON CONFLICT(user_id,"+column+") WHERE "+column+" IS NOT NULL DO UPDATE SET is_like=excluded.is_like", s.UserID, id, like)
	if err != nil {
		serverError(w, err)
		return
	}
	var likes, dislikes int
	if err := tx.QueryRow("SELECT COALESCE(SUM(is_like=1),0),COALESCE(SUM(is_like=0),0) FROM likes WHERE "+column+"=?", id).Scan(&likes, &dislikes); err != nil {
		serverError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	if kind == "post" {
		notifyPostOwner(id)
	}
	result := map[string]interface{}{"kind": kind, "id": id, "likes": likes, "dislikes": dislikes}
	broadcastMessage(Message{Type: "reaction_updated", Content: result})
	writeJSON(w, http.StatusOK, result)
}
