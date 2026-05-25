package handlers

import (
	"RTF/database"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
)

func ServeReports(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		if !isAdmin(s.UserID) {
			http.Error(w, "Moderator access required", http.StatusForbidden)
			return
		}
		limit, offset, err := pagination(r)
		if err != nil {
			http.Error(w, "Invalid pagination", http.StatusBadRequest)
			return
		}
		rows, err := database.DBInstance.DB.Query(`SELECT r.id,r.post_id,r.reason,r.created_at,u.username,p.title FROM reports r JOIN users u ON u.id=r.user_id JOIN posts p ON p.id=r.post_id ORDER BY r.id DESC LIMIT ? OFFSET ?`, limit, offset)
		if err != nil {
			serverError(w, err)
			return
		}
		defer rows.Close()
		type report struct {
			ID, PostID      int
			Reason          string
			CreatedAt       time.Time
			Username, Title string
		}
		reports := []map[string]interface{}{}
		for rows.Next() {
			var item report
			if err := rows.Scan(&item.ID, &item.PostID, &item.Reason, &item.CreatedAt, &item.Username, &item.Title); err != nil {
				serverError(w, err)
				return
			}
			reports = append(reports, map[string]interface{}{"id": item.ID, "post_id": item.PostID, "reason": item.Reason, "created_at": item.CreatedAt, "username": item.Username, "title": item.Title})
		}
		if err := rows.Err(); err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, reports)
		return
	}
	var input struct {
		PostID int    `json:"post_id"`
		Reason string `json:"reason"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Reason = strings.TrimSpace(input.Reason)
	if input.PostID < 1 || input.Reason == "" || len([]rune(input.Reason)) > 500 {
		http.Error(w, "Use a valid post ID and a reason of 1–500 characters", http.StatusBadRequest)
		return
	}
	result, err := database.DBInstance.DB.Exec(`INSERT INTO reports(user_id,post_id,reason) SELECT ?,?,? WHERE EXISTS(SELECT 1 FROM posts WHERE id=?) ON CONFLICT(user_id,post_id) DO UPDATE SET reason=excluded.reason`, s.UserID, input.PostID, input.Reason, input.PostID)
	if err != nil {
		serverError(w, err)
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "Report submitted"})
}
func ServeModerate(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	if !isAdmin(s.UserID) {
		http.Error(w, "Moderator access required", http.StatusForbidden)
		return
	}
	var input struct {
		ReportID int    `json:"report_id"`
		Action   string `json:"action"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.ReportID < 1 || (input.Action != "dismiss" && input.Action != "delete") {
		http.Error(w, "Choose dismiss or delete for a valid report", http.StatusBadRequest)
		return
	}
	tx, err := database.DBInstance.DB.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	var postID int
	var reason string
	err = tx.QueryRow("SELECT post_id,reason FROM reports WHERE id=?", input.ReportID).Scan(&postID, &reason)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Report not found", http.StatusNotFound)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if input.Action == "delete" {
		_, err = tx.Exec("DELETE FROM posts WHERE id=?", postID)
	} else {
		_, err = tx.Exec("DELETE FROM reports WHERE id=?", input.ReportID)
	}
	if err != nil {
		serverError(w, err)
		return
	}
	_, err = tx.Exec("INSERT INTO moderation_actions(moderator_id,post_id,action,reason) VALUES(?,?,?,?)", s.UserID, postID, input.Action, reason)
	if err != nil {
		serverError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	if input.Action == "delete" {
		broadcastMessage(Message{Type: "post_deleted", Content: map[string]interface{}{"post_id": postID}})
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Report resolved"})
}
