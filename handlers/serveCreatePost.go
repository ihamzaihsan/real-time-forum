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

	// Add character limit validation
    if len(post.Title) > 100 {
        http.Error(w, "Title must be 100 characters or less", http.StatusBadRequest)
        return
    }

    if len(post.Content) > 5000 {
        http.Error(w, "Content must be 5000 characters or less", http.StatusBadRequest)
        return
    }
	
	// Validate categories
	if len(post.Categories) == 0 {
		http.Error(w, "At least one category is required", http.StatusBadRequest)
		return
	}

	userID := getUserIDFromSession(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	post.CreatedAt = time.Now()

	result, err := database.DBInstance.DB.Exec(
		"INSERT INTO posts (title, content, user_id, created_at) VALUES (?, ?, ?, ?)",
		post.Title, post.Content, userID, post.CreatedAt,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Database error 2: %v", err)
		return
	}

	postID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Database error 3: %v", err)
		return
	}

	for _, category := range post.Categories {
		_, err := database.DBInstance.DB.Exec(
			"INSERT INTO post_categories (post_id, category_id) VALUES (?, (SELECT id FROM categories WHERE name = ?))",
			postID, category,
		)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			log.Printf("Database error 4: %v", err)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Post created successfully",
		"post_id": postID,
	})
}
