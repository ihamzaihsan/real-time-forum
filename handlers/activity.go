package handlers

import (
	"RTF/database"
	"net/http"
)

func notifyPostOwner(postID int) {
	var owner int
	if err := database.DBInstance.DB.QueryRow("SELECT user_id FROM posts WHERE id=? AND status='published'", postID).Scan(&owner); err == nil {
		sendToUser(owner, Message{Type: "notification_changed", Content: nil})
	}
}

func ServeActivity(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		http.Error(w, "Invalid pagination", 400)
		return
	}
	result := map[string]interface{}{}
	queries := []struct {
		key, query string
		args       []interface{}
	}{
		{"posts", `SELECT p.id,p.title,p.content,p.status FROM posts p WHERE p.user_id=? ORDER BY p.id DESC LIMIT ? OFFSET ?`, []interface{}{s.UserID, limit, offset}},
		{"comments", `SELECT c.id,c.content,c.status,CASE WHEN ` + visibility("p", r) + ` THEN p.id ELSE NULL END AS post_id,CASE WHEN ` + visibility("p", r) + ` THEN p.title ELSE 'Discussion awaiting review' END AS title FROM comments c JOIN posts p ON p.id=c.post_id WHERE c.user_id=? ORDER BY c.id DESC LIMIT ? OFFSET ?`, []interface{}{s.UserID, limit, offset}},
		{"reactions", `SELECT l.id,l.is_like,p.id AS post_id,p.title,l.comment_id,CASE WHEN l.comment_id IS NOT NULL THEN c.content ELSE '' END AS comment FROM likes l LEFT JOIN comments c ON c.id=l.comment_id JOIN posts p ON p.id=COALESCE(l.post_id,c.post_id) WHERE l.user_id=? AND ` + visibility("p", r) + ` AND (l.comment_id IS NULL OR ` + visibility("c", r) + `) ORDER BY l.id DESC LIMIT ? OFFSET ?`, []interface{}{s.UserID, limit, offset}},
	}
	for _, q := range queries {
		items, err := queryItems(q.query, q.args...)
		if err != nil {
			serverError(w, err)
			return
		}
		result[q.key] = items
	}
	writeJSON(w, 200, result)
}

func ServeNotifications(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		var input struct {
			ID int `json:"id"`
		}
		if !decodeJSON(w, r, &input) {
			return
		}
		if input.ID < 0 {
			http.Error(w, "Invalid notification ID", 400)
			return
		}
		if _, err := database.DBInstance.DB.Exec("UPDATE notifications SET is_read=1 WHERE recipient_id=? AND (?=0 OR id=?)", s.UserID, input.ID, input.ID); err != nil {
			serverError(w, err)
			return
		}
		sendToUser(s.UserID, Message{Type: "notification_changed", Content: nil})
		w.WriteHeader(204)
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		http.Error(w, "Invalid pagination", 400)
		return
	}
	items, err := queryItems(`SELECT n.id,n.post_id,n.comment_id,n.kind,n.is_read,n.created_at,u.username AS actor,p.title FROM notifications n JOIN users u ON u.id=n.actor_id JOIN posts p ON p.id=n.post_id LEFT JOIN comments c ON c.id=n.comment_id WHERE n.recipient_id=? AND p.status='published' AND (n.comment_id IS NULL OR c.status='published') ORDER BY n.id DESC LIMIT ? OFFSET ?`, s.UserID, limit, offset)
	if err != nil {
		serverError(w, err)
		return
	}
	var unread int
	if err := database.DBInstance.DB.QueryRow(`SELECT COUNT(*) FROM notifications n JOIN posts p ON p.id=n.post_id LEFT JOIN comments c ON c.id=n.comment_id WHERE n.recipient_id=? AND n.is_read=0 AND p.status='published' AND (n.comment_id IS NULL OR c.status='published')`, s.UserID).Scan(&unread); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 200, map[string]interface{}{"items": items, "unread": unread})
}
