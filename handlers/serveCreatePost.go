package handlers

import (
	"RTF/database"
	"RTF/models"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

func ServeCreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		http.ServeFile(w, r, filepath.Join("frontend", "index.html"))
		return
	}

	var post models.Post
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	// Add after json.NewDecoder
	log.Printf("Received post data: %+v", post)

	// Get user_id from username
	var userID int
	err := database.DBInstance.DB.QueryRow("SELECT id FROM users WHERE username = ?", post.Username).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	post.CreatedAt = time.Now()

	result, err := database.DBInstance.DB.Exec(
		"INSERT INTO posts (title, content, user_id, created_at) VALUES (?, ?, ?, ?)",
		post.Title, post.Content, userID, post.CreatedAt,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	postID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	for _, category := range post.Categories {
		_, err := database.DBInstance.DB.Exec(
			"INSERT INTO post_categories (post_id, category_id) VALUES (?, (SELECT id FROM categories WHERE name = ?))",
			postID, category,
		)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Post created successfully",
		"post_id": postID,
	})
}
