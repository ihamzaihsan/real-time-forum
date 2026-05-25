package handlers

import (
	"RTF/database"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
)

type messageRecord struct {
	ID         int64     `json:"id"`
	SenderID   int       `json:"sender_id"`
	ReceiverID int       `json:"receiver_id"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	IsRead     bool      `json:"is_read"`
	Sender     string    `json:"sender,omitempty"`
	ClientID   string    `json:"client_id,omitempty"`
}

func ServeMessages(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	id, err := positiveID(strings.TrimPrefix(r.URL.Path, "/messages/"))
	if err != nil || id == s.UserID {
		http.Error(w, "Choose another user", http.StatusBadRequest)
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		http.Error(w, "Invalid pagination", http.StatusBadRequest)
		return
	}
	before := 0
	if value := r.URL.Query().Get("before"); value != "" {
		before, err = positiveID(value)
		if err != nil || offset != 0 {
			http.Error(w, "Use a positive before ID without offset", http.StatusBadRequest)
			return
		}
	}
	var found int
	err = database.DBInstance.DB.QueryRow("SELECT id FROM users WHERE id=?", id).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	rows, err := database.DBInstance.DB.Query(`SELECT id,sender_id,receiver_id,content,created_at,is_read FROM messages
 WHERE ((sender_id=? AND receiver_id=?) OR (sender_id=? AND receiver_id=?)) AND (?=0 OR id<?) ORDER BY id DESC LIMIT ? OFFSET ?`, s.UserID, id, id, s.UserID, before, before, limit, offset)
	if err != nil {
		serverError(w, err)
		return
	}
	messages := make([]messageRecord, 0)
	for rows.Next() {
		var m messageRecord
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Content, &m.CreatedAt, &m.IsRead); err != nil {
			rows.Close()
			serverError(w, err)
			return
		}
		messages = append(messages, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		serverError(w, err)
		return
	}
	if len(messages) > 0 {
		_, err = database.DBInstance.DB.Exec("UPDATE messages SET is_read=TRUE WHERE receiver_id=? AND sender_id=? AND id>=? AND id<=?", s.UserID, id, messages[len(messages)-1].ID, messages[0].ID)
		if err != nil {
			serverError(w, err)
			return
		}
		for i := range messages {
			if messages[i].ReceiverID == s.UserID {
				messages[i].IsRead = true
			}
		}
	}
	writeJSON(w, http.StatusOK, messages)
}
