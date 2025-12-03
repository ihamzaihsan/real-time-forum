package handlers

import (
	"RTF/database"
	"RTF/models"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func ServeCreateComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var comment models.Comment
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	comment.Content = strings.TrimSpace(comment.Content)
	if comment.Content == "" {
		http.Error(w, "Comment cannot be empty", http.StatusBadRequest)
		return
	}
	if len([]rune(comment.Content)) > 200 {
		http.Error(w, "Comment must be 200 characters or less", http.StatusBadRequest)
		return
	}

	userID := getUserIDFromSession(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	comment.UserID = userID
	comment.CreatedAt = time.Now()

	result, err := database.DBInstance.DB.Exec(
		"INSERT INTO comments (content, user_id, post_id, created_at) VALUES (?, ?, ?, ?)",
		comment.Content, comment.UserID, comment.PostID, comment.CreatedAt,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	commentID, _ := result.LastInsertId()
	comment.ID = int(commentID)

	go broadcastComment(comment)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(comment)
}

func ServeGetComments(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	postID := r.URL.Query().Get("post_id")
	if postID == "" {
		http.Error(w, "Post ID required", http.StatusBadRequest)
		return
	}

	rows, err := database.DBInstance.DB.Query(`
        SELECT c.id, c.content, c.user_id, u.username, c.created_at,
        (SELECT COUNT(*) FROM likes WHERE comment_id = c.id AND is_like = 1) as likes,
        (SELECT COUNT(*) FROM likes WHERE comment_id = c.id AND is_like = 0) as dislikes
        FROM comments c
        JOIN users u ON c.user_id = u.id
        WHERE c.post_id = ?
        ORDER BY c.created_at DESC
    `, postID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	comments := make([]models.Comment, 0)
	for rows.Next() {
		var comment models.Comment
		err := rows.Scan(
			&comment.ID, &comment.Content, &comment.UserID,
			&comment.Username, &comment.CreatedAt,
			&comment.Likes, &comment.Dislikes,
		)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		comments = append(comments, comment)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func broadcastComment(comment models.Comment) {
	message := Message{
		Type:    "new_comment",
		Content: comment,
	}

	broadcastMessage(message)
}
