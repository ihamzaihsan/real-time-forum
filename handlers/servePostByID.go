package handlers

import (
	"RTF/database"
	"RTF/models"
	"encoding/json"
	"net/http"
	"strings"
)

func ServePostByID(w http.ResponseWriter, r *http.Request) {

    postID := strings.TrimPrefix(r.URL.Path, "/post/")

	rows, err := database.DBInstance.DB.Query(`
        SELECT p.id, p.title, p.content, u.username, p.created_at,
        (SELECT COUNT(*) FROM likes WHERE post_id = p.id AND is_like = 1) AS likes,
        (SELECT COUNT(*) FROM likes WHERE post_id = p.id AND is_like = 0) AS dislikes,
        COALESCE(GROUP_CONCAT(c.name), '') AS categories
        FROM posts p
        JOIN users u ON p.user_id = u.id
        LEFT JOIN post_categories pc ON p.id = pc.post_id
        LEFT JOIN categories c ON pc.category_id = c.id
        WHERE p.id = ?
        GROUP BY p.id
    `, postID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var post models.Post
	var categoriesStr string

	if !rows.Next() {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.Username, &post.CreatedAt, &post.Likes, &post.Dislikes, &categoriesStr); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if categoriesStr != "" {
		post.Categories = strings.Split(categoriesStr, ",")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(post)
}
