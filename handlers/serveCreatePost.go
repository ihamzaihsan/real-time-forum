package handlers

import (
	"RTF/database"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
)

func ServeCreatePost(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}
	if r.Method == http.MethodGet {
		serveShell(w, r)
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	var input struct {
		Title      string   `json:"title"`
		Content    string   `json:"content"`
		Categories []string `json:"categories"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	if input.Title == "" || len([]rune(input.Title)) > 200 || input.Content == "" || len([]rune(input.Content)) > 2000 {
		http.Error(w, "Use a title of 1–200 characters and content of 1–2,000 characters", http.StatusBadRequest)
		return
	}
	if len(input.Categories) < 1 || len(input.Categories) > 5 {
		http.Error(w, "Choose 1–5 distinct categories", http.StatusBadRequest)
		return
	}
	tx, err := database.DBInstance.DB.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	categoryIDs := []int{}
	seen := map[string]bool{}
	for _, category := range input.Categories {
		if seen[category] {
			http.Error(w, "Duplicate category", http.StatusBadRequest)
			return
		}
		seen[category] = true
		var id int
		err := tx.QueryRow("SELECT id FROM categories WHERE name=?", category).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Unknown category", http.StatusBadRequest)
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		categoryIDs = append(categoryIDs, id)
	}
	result, err := tx.Exec("INSERT INTO posts(title,content,user_id,created_at) VALUES(?,?,?,?)", input.Title, input.Content, s.UserID, time.Now().UTC())
	if err != nil {
		serverError(w, err)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		serverError(w, err)
		return
	}
	for _, categoryID := range categoryIDs {
		if _, err := tx.Exec("INSERT INTO post_categories(post_id,category_id) VALUES(?,?)", id, categoryID); err != nil {
			serverError(w, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	broadcastMessage(Message{Type: "new_post", Content: map[string]interface{}{"post_id": id}})
	writeJSON(w, http.StatusCreated, map[string]interface{}{"post_id": id})
}
