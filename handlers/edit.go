package handlers

import (
	"RTF/database"
	"database/sql"
	"errors"
	"net/http"
	"strings"
)

func ServeEditPost(w http.ResponseWriter, r *http.Request)    { editContent(w, r, "post") }
func ServeEditComment(w http.ResponseWriter, r *http.Request) { editContent(w, r, "comment") }
func editContent(w http.ResponseWriter, r *http.Request, kind string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	var input struct {
		ID         int      `json:"id"`
		Title      string   `json:"title"`
		Content    string   `json:"content"`
		Categories []string `json:"categories"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	max := 200
	if kind == "post" {
		max = 2000
	}
	if input.ID < 1 || input.Content == "" || len([]rune(input.Content)) > max || (kind == "post" && (input.Title == "" || len([]rune(input.Title)) > 200 || len(input.Categories) < 1 || len(input.Categories) > 5)) {
		http.Error(w, "Enter valid text and topics within the displayed limits", 400)
		return
	}
	tx, err := database.DBInstance.DB.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	var owner int
	var oldStatus string
	if err = tx.QueryRow("SELECT user_id,status FROM "+kind+"s WHERE id=?", input.ID).Scan(&owner, &oldStatus); err != nil {
		mutationError(w, err)
		return
	}
	if owner != s.UserID {
		http.Error(w, "Only the author can edit this content", 403)
		return
	}
	status, err := submissionStatus(tx, s.UserID)
	if err != nil {
		serverError(w, err)
		return
	}
	if oldStatus == "pending" {
		status = "pending"
	}
	postID := input.ID
	if kind == "post" {
		seen := map[string]bool{}
		ids := []int{}
		for _, name := range input.Categories {
			if seen[name] {
				http.Error(w, "Duplicate topic", 400)
				return
			}
			seen[name] = true
			var id int
			if err = tx.QueryRow("SELECT id FROM categories WHERE name=?", name).Scan(&id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					http.Error(w, "Unknown topic", 400)
				} else {
					serverError(w, err)
				}
				return
			}
			ids = append(ids, id)
		}
		if _, err = tx.Exec("UPDATE posts SET title=?,content=?,status=? WHERE id=?", input.Title, input.Content, status, input.ID); err != nil {
			serverError(w, err)
			return
		}
		if _, err = tx.Exec("DELETE FROM post_categories WHERE post_id=?", input.ID); err != nil {
			serverError(w, err)
			return
		}
		for _, id := range ids {
			if _, err = tx.Exec("INSERT INTO post_categories(post_id,category_id) VALUES(?,?)", input.ID, id); err != nil {
				serverError(w, err)
				return
			}
		}
	} else {
		if err = tx.QueryRow("SELECT post_id FROM comments WHERE id=?", input.ID).Scan(&postID); err != nil {
			serverError(w, err)
			return
		}
		if _, err = tx.Exec("UPDATE comments SET content=?,status=? WHERE id=?", input.Content, status, input.ID); err != nil {
			serverError(w, err)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	contentChanged()
	communityChanged()
	notifyPostOwner(postID)
	writeJSON(w, 200, map[string]interface{}{"id": input.ID, "status": status})
}

func ServeDeleteComment(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	var input struct {
		ID int `json:"id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.ID < 1 {
		http.Error(w, "Invalid comment ID", 400)
		return
	}
	tx, err := database.DBInstance.DB.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	var owner, postID int
	if err = tx.QueryRow("SELECT user_id,post_id FROM comments WHERE id=?", input.ID).Scan(&owner, &postID); err != nil {
		mutationError(w, err)
		return
	}
	role, err := roleIn(tx, s.UserID)
	if err != nil {
		serverError(w, err)
		return
	}
	if owner != s.UserID && role != "admin" {
		http.Error(w, "Only the author or administrator can delete this reply", 403)
		return
	}
	if _, err = tx.Exec("DELETE FROM comments WHERE id=?", input.ID); err != nil {
		serverError(w, err)
		return
	}
	if owner != s.UserID {
		if _, err = tx.Exec("INSERT INTO moderation_actions(moderator_id,post_id,action,reason) VALUES(?,?, 'delete_comment',?)", s.UserID, postID, "Staff deletion"); err != nil {
			serverError(w, err)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	contentChanged()
	communityChanged()
	notifyPostOwner(postID)
	w.WriteHeader(204)
}
