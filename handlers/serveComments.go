package handlers

import (
	"RTF/database"
	"RTF/models"
	"net/http"
	"strings"
	"time"
)

func ServeCreateComment(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	var input struct {
		PostID  int    `json:"post_id"`
		Content string `json:"content"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Content = strings.TrimSpace(input.Content)
	if input.PostID < 1 || input.Content == "" || len([]rune(input.Content)) > 200 {
		http.Error(w, "Use a valid post ID and a reply of 1–200 characters", http.StatusBadRequest)
		return
	}
	created := time.Now().UTC()
	result, err := database.DBInstance.DB.Exec("INSERT INTO comments(content,user_id,post_id,created_at) SELECT ?,?,?,? WHERE EXISTS(SELECT 1 FROM posts WHERE id=?)", input.Content, s.UserID, input.PostID, created, input.PostID)
	if err != nil {
		serverError(w, err)
		return
	}
	if count, err := result.RowsAffected(); err != nil {
		serverError(w, err)
		return
	} else if count == 0 {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		serverError(w, err)
		return
	}
	comment := models.Comment{ID: int(id), PostID: input.PostID, UserID: s.UserID, Username: s.Username, Content: input.Content, CreatedAt: created}
	broadcastMessage(Message{Type: "new_comment", Content: comment})
	writeJSON(w, http.StatusCreated, comment)
}
func ServeGetComments(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	id, err := positiveID(r.URL.Query().Get("post_id"))
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}
	rows, err := database.DBInstance.DB.Query(`SELECT c.id,c.post_id,c.content,c.user_id,u.username,c.created_at,
 (SELECT COUNT(*) FROM likes WHERE comment_id=c.id AND is_like=1),
 (SELECT COUNT(*) FROM likes WHERE comment_id=c.id AND is_like=0)
 FROM comments c JOIN users u ON u.id=c.user_id WHERE c.post_id=? ORDER BY julianday(c.created_at) DESC,c.id DESC`, id)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	comments := make([]models.Comment, 0)
	for rows.Next() {
		var c models.Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.Content, &c.UserID, &c.Username, &c.CreatedAt, &c.Likes, &c.Dislikes); err != nil {
			serverError(w, err)
			return
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, comments)
}
