package handlers

import (
	"RTF/database"
	"database/sql"
	"errors"
	"mime/multipart"
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
	var attachment *multipart.FileHeader
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		err := r.ParseMultipartForm(1 << 20)
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
		if err != nil {
			http.Error(w, "Invalid upload; images must be at most 20 MiB", http.StatusRequestEntityTooLarge)
			return
		}
		input.Title, input.Content, input.Categories = r.FormValue("title"), r.FormValue("content"), r.MultipartForm.Value["categories"]
		for field, files := range r.MultipartForm.File {
			if field != "image" || len(files) != 1 {
				http.Error(w, "Supply at most one image", 400)
				return
			}
			attachment = files[0]
		}
	} else if !decodeJSON(w, r, &input) {
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
	imageName, err := saveImage(attachment)
	if err != nil {
		var invalid *imageInputError
		if errors.As(err, &invalid) {
			http.Error(w, invalid.Error(), http.StatusBadRequest)
		} else if errors.Is(err, errUploadBusy) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, err.Error(), http.StatusTooManyRequests)
		} else {
			serverError(w, err)
		}
		return
	}
	saved := false
	defer func() {
		if !saved && imageName != "" {
			removeUpload(imageName)
		}
	}()
	status, err := submissionStatus(tx, s.UserID)
	if err != nil {
		serverError(w, err)
		return
	}
	result, err := tx.Exec("INSERT INTO posts(title,content,user_id,created_at,image_path,status) VALUES(?,?,?,?,?,?)", input.Title, input.Content, s.UserID, time.Now().UTC(), imageName, status)
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
	saved = true
	if status == "published" {
		broadcastMessage(Message{Type: "new_post", Content: map[string]interface{}{"post_id": id}})
	}
	communityChanged()
	writeJSON(w, http.StatusCreated, map[string]interface{}{"post_id": id, "status": status})
}
