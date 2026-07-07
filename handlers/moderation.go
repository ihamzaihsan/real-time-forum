package handlers

import (
	"RTF/database"
	"database/sql"
	"errors"
	"github.com/mattn/go-sqlite3"
	"net/http"
	"os"
	"strings"
)

func queryItems(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := database.DBInstance.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	items := []map[string]interface{}{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		targets := make([]interface{}, len(columns))
		for i := range values {
			targets[i] = &values[i]
		}
		if err := rows.Scan(targets...); err != nil {
			return nil, err
		}
		item := map[string]interface{}{}
		for i, name := range columns {
			if b, ok := values[i].([]byte); ok {
				item[name] = string(b)
			} else {
				item[name] = values[i]
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func mutationError(w http.ResponseWriter, err error) {
	var e sqlite3.Error
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Item not found or no longer available", 404)
	} else if errors.As(err, &e) && e.Code == sqlite3.ErrConstraint {
		http.Error(w, "Duplicate pending request/report, invalid topic, or conflicting operation", 409)
	} else {
		serverError(w, err)
	}
}

func ServeCommunity(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		role := userRole(s.UserID)
		limit, offset, err := pagination(r)
		if err != nil {
			http.Error(w, "Invalid pagination", 400)
			return
		}
		result := map[string]interface{}{"role": role, "premoderate": os.Getenv("FORUM_PREMODERATE") == "true"}
		clauses := []struct {
			key, query string
			args       []interface{}
		}{
			{"requests", `SELECT m.*,u.username FROM moderator_requests m JOIN users u ON u.id=m.user_id WHERE m.user_id=? OR ? ORDER BY m.id DESC LIMIT ? OFFSET ?`, []interface{}{s.UserID, role == "admin", limit, offset}},
			{"reports", `SELECT r.*,u.username FROM reports r JOIN users u ON u.id=r.user_id WHERE r.user_id=? OR ? ORDER BY r.id DESC LIMIT ? OFFSET ?`, []interface{}{s.UserID, role == "admin", limit, offset}},
		}
		if staffRole(role) {
			clauses = append(clauses,
				struct {
					key, query string
					args       []interface{}
				}{"pending_posts", `SELECT id,title,content,user_id,COALESCE(image_path,'') AS image_path FROM posts WHERE status='pending' ORDER BY id LIMIT ? OFFSET ?`, []interface{}{limit, offset}},
				struct {
					key, query string
					args       []interface{}
				}{"pending_comments", `SELECT c.id,c.content,c.post_id,p.title FROM comments c JOIN posts p ON p.id=c.post_id WHERE c.status='pending' ORDER BY c.id LIMIT ? OFFSET ?`, []interface{}{limit, offset}})
		}
		topics, err := queryItems("SELECT id,name FROM categories ORDER BY name")
		if err != nil {
			serverError(w, err)
			return
		}
		result["topics"] = topics
		if role == "admin" {
			clauses = append(clauses, struct {
				key, query string
				args       []interface{}
			}{"users", `SELECT id,username,role FROM users ORDER BY id LIMIT ? OFFSET ?`, []interface{}{limit, offset}})
		}
		for _, item := range clauses {
			entries, err := queryItems(item.query, item.args...)
			if err != nil {
				serverError(w, err)
				return
			}
			result[item.key] = entries
		}
		writeJSON(w, 200, result)
		return
	}
	var input struct {
		Action  string `json:"action"`
		ID      int    `json:"id"`
		Message string `json:"message"`
		Reply   string `json:"reply"`
		Role    string `json:"role"`
		Name    string `json:"name"`
		Kind    string `json:"kind"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Message = strings.TrimSpace(input.Message)
	input.Reply = strings.TrimSpace(input.Reply)
	input.Name = strings.TrimSpace(input.Name)
	if len([]rune(input.Message)) > 2000 || len([]rune(input.Reply)) > 2000 {
		http.Error(w, "Message/reply must be at most 2,000 characters", 400)
		return
	}
	tx, err := database.DBInstance.DB.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	role, err := roleIn(tx, s.UserID)
	if err != nil {
		serverError(w, err)
		return
	}
	var imageName string
	affectedUser := 0
	switch input.Action {
	case "request":
		if role != "member" || input.Message == "" {
			http.Error(w, "Members may submit a request with a message", 400)
			return
		}
		_, err = tx.Exec("INSERT INTO moderator_requests(user_id,message) VALUES(?,?)", s.UserID, input.Message)
	case "accept", "decline":
		if role != "admin" {
			http.Error(w, "Administrator access required", 403)
			return
		}
		if input.Reply == "" {
			http.Error(w, "Enter a reply", 400)
			return
		}
		err = tx.QueryRow("SELECT user_id FROM moderator_requests WHERE id=? AND status='pending'", input.ID).Scan(&affectedUser)
		if err != nil {
			mutationError(w, err)
			return
		}
		status := "declined"
		if input.Action == "accept" {
			status = "accepted"
			_, err = tx.Exec("UPDATE users SET role='moderator' WHERE id=? AND role='member'", affectedUser)
			if err != nil {
				mutationError(w, err)
				return
			}
		}
		_, err = tx.Exec("UPDATE moderator_requests SET status=?,reply=? WHERE id=?", status, input.Reply, input.ID)
	case "role":
		if role != "admin" {
			http.Error(w, "Administrator access required", 403)
			return
		}
		if input.Role != "member" && input.Role != "moderator" {
			http.Error(w, "Choose member or moderator", 400)
			return
		}
		targetRole, e := roleIn(tx, input.ID)
		if e != nil {
			mutationError(w, e)
			return
		}
		if targetRole == "admin" {
			http.Error(w, "Administrator roles require local access", 403)
			return
		}
		_, err = tx.Exec("UPDATE users SET role=? WHERE id=?", input.Role, input.ID)
		affectedUser = input.ID
		if err == nil && input.Role == "moderator" {
			_, err = tx.Exec("UPDATE moderator_requests SET status='accepted',reply='Promoted by administrator' WHERE user_id=? AND status='pending'", input.ID)
		}
	case "topic_create", "topic_delete":
		if role != "admin" {
			http.Error(w, "Administrator access required", 403)
			return
		}
		if input.Action == "topic_create" {
			if input.Name == "" || len(input.Name) > 32 || strings.ContainsAny(input.Name, ",\r\n") {
				http.Error(w, "Use a topic of 1-32 bytes without commas or newlines", 400)
				return
			}
			_, err = tx.Exec("INSERT INTO categories(name) VALUES(?)", input.Name)
		} else {
			var count int
			err = tx.QueryRow("SELECT COUNT(*) FROM post_categories WHERE category_id=?", input.ID).Scan(&count)
			if err != nil {
				serverError(w, err)
				return
			}
			if count > 0 {
				http.Error(w, "Topic is used by discussions and cannot be deleted", 409)
				return
			}
			result, e := tx.Exec("DELETE FROM categories WHERE id=?", input.ID)
			err = e
			if err == nil {
				n, _ := result.RowsAffected()
				if n == 0 {
					err = sql.ErrNoRows
				}
			}
		}
	case "approve", "reject":
		if !staffRole(role) {
			http.Error(w, "Staff access required", 403)
			return
		}
		if input.Kind != "post" && input.Kind != "comment" {
			http.Error(w, "Choose post or comment", 400)
			return
		}
		table := input.Kind + "s"
		var owner int
		err = tx.QueryRow("SELECT user_id FROM "+table+" WHERE id=? AND status='pending'", input.ID).Scan(&owner)
		if err != nil {
			mutationError(w, err)
			return
		}
		if input.Action == "approve" {
			if input.Kind == "comment" {
				var published bool
				err = tx.QueryRow("SELECT p.status='published' FROM posts p JOIN comments c ON c.post_id=p.id WHERE c.id=?", input.ID).Scan(&published)
				if err != nil {
					serverError(w, err)
					return
				}
				if !published {
					http.Error(w, "Approve the parent discussion first", 409)
					return
				}
			}
			_, err = tx.Exec("UPDATE "+table+" SET status='published' WHERE id=?", input.ID)
		} else {
			if input.Kind == "post" {
				if err = tx.QueryRow("SELECT COALESCE(image_path,'') FROM posts WHERE id=?", input.ID).Scan(&imageName); err != nil {
					serverError(w, err)
					return
				}
			}
			_, err = tx.Exec("DELETE FROM "+table+" WHERE id=?", input.ID)
		}
		if err == nil {
			_, err = tx.Exec("INSERT INTO moderation_actions(moderator_id,post_id,action,reason) VALUES(?,?,?,?)", s.UserID, input.ID, input.Action+"_"+input.Kind, input.Reply)
		}
	default:
		http.Error(w, "Unknown community action", 400)
		return
	}
	if err != nil {
		mutationError(w, err)
		return
	}
	if input.Action != "request" && input.Action != "approve" && input.Action != "reject" {
		targetKind := "user"
		if input.Action == "accept" || input.Action == "decline" {
			targetKind = "request"
		}
		if strings.HasPrefix(input.Action, "topic_") {
			targetKind = "topic"
		}
		reason := input.Reply
		if input.Action == "role" {
			reason = input.Role
		}
		if input.Action == "topic_create" {
			reason = input.Name
		}
		if _, err = tx.Exec("INSERT INTO moderation_actions(moderator_id,post_id,action,reason,target_kind,target_id) VALUES(?,0,?,?,?,?)", s.UserID, input.Action, reason, targetKind, input.ID); err != nil {
			serverError(w, err)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	cleanupImage(imageName)
	if input.Action == "approve" && input.Kind == "comment" {
		var postID int
		if err := database.DBInstance.DB.QueryRow("SELECT post_id FROM comments WHERE id=?", input.ID).Scan(&postID); err == nil {
			notifyPostOwner(postID)
		}
	}
	if affectedUser > 0 {
		sendToUser(affectedUser, Message{Type: "account_changed", Content: nil})
	}
	if strings.HasPrefix(input.Action, "topic_") {
		broadcastMessage(Message{Type: "topics_changed", Content: nil})
	}
	contentChanged()
	communityChanged()
	writeJSON(w, 200, map[string]string{"message": "Saved"})
}

func ServeReports(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		role := userRole(s.UserID)
		limit, offset, err := pagination(r)
		if err != nil {
			http.Error(w, "Invalid pagination", 400)
			return
		}
		items, err := queryItems(`SELECT r.*,u.username FROM reports r JOIN users u ON u.id=r.user_id WHERE r.user_id=? OR ? ORDER BY r.id DESC LIMIT ? OFFSET ?`, s.UserID, role == "admin", limit, offset)
		if err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, 200, items)
		return
	}
	var input struct {
		PostID   int    `json:"post_id"`
		Reason   string `json:"reason"`
		Category string `json:"category"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Reason = strings.TrimSpace(input.Reason)
	if input.PostID < 1 || input.Reason == "" || len([]rune(input.Reason)) > 2000 {
		http.Error(w, "Enter a discussion and a reason of 1-2,000 characters", 400)
		return
	}
	if input.Category == "" {
		input.Category = "other"
	}
	if input.Category != "other" && input.Category != "irrelevant" && input.Category != "obscene" && input.Category != "illegal" && input.Category != "insulting" {
		http.Error(w, "Invalid report category", 400)
		return
	}
	tx, err := database.DBInstance.DB.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	role, err := roleIn(tx, s.UserID)
	if err != nil {
		serverError(w, err)
		return
	}
	// Member reporting from the existing forum is retained, but members cannot report hidden content.
	var title string
	query := "SELECT title FROM posts WHERE id=? AND status='published'"
	if staffRole(role) {
		query = "SELECT title FROM posts WHERE id=?"
	}
	if err = tx.QueryRow(query, input.PostID).Scan(&title); err != nil {
		mutationError(w, err)
		return
	}
	_, err = tx.Exec("INSERT INTO reports(user_id,post_id,original_post_id,title,reason,category) VALUES(?,?,?,?,?,?)", s.UserID, input.PostID, input.PostID, title, input.Reason, input.Category)
	if err != nil {
		mutationError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	communityChanged()
	writeJSON(w, 201, map[string]string{"message": "Report submitted"})
}

func ServeModerate(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	var input struct {
		ReportID int    `json:"report_id"`
		Action   string `json:"action"`
		Reply    string `json:"reply"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Reply = strings.TrimSpace(input.Reply)
	if input.ReportID < 1 || (input.Action != "dismiss" && input.Action != "delete" && input.Action != "reply") || input.Reply == "" || len([]rune(input.Reply)) > 2000 {
		http.Error(w, "Choose an action and enter a reply of 1-2,000 characters", 400)
		return
	}
	tx, err := database.DBInstance.DB.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	role, err := roleIn(tx, s.UserID)
	if err != nil {
		serverError(w, err)
		return
	}
	if role != "admin" {
		http.Error(w, "Administrator access required", 403)
		return
	}
	var postID, reporter int
	var imageName string
	err = tx.QueryRow("SELECT original_post_id,user_id FROM reports WHERE id=? AND answered=0", input.ReportID).Scan(&postID, &reporter)
	if err != nil {
		mutationError(w, err)
		return
	}
	if input.Action == "delete" {
		err = tx.QueryRow("SELECT COALESCE(image_path,'') FROM posts WHERE id=?", postID).Scan(&imageName)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			serverError(w, err)
			return
		}
		if _, err = tx.Exec("DELETE FROM posts WHERE id=?", postID); err != nil {
			serverError(w, err)
			return
		}
	}
	if _, err = tx.Exec("UPDATE reports SET reply=?,answered=1 WHERE id=?", input.Reply, input.ReportID); err != nil {
		serverError(w, err)
		return
	}
	if _, err = tx.Exec("INSERT INTO moderation_actions(moderator_id,post_id,action,reason) VALUES(?,?,?,?)", s.UserID, postID, input.Action, input.Reply); err != nil {
		serverError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	cleanupImage(imageName)
	sendToUser(reporter, Message{Type: "community_changed", Content: nil})
	communityChanged()
	contentChanged()
	writeJSON(w, 200, map[string]string{"message": "Report answered"})
}
