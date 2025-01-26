package handlers

import (
	"RTF/database"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func ServeMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	userId, err := strconv.Atoi(pathParts[len(pathParts)-1])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get pagination parameters
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 10 // Default limit
	}

	// Get current user ID from session
	currentUserId := getUserIDFromSession(r)
	if currentUserId == 0 {
		// Add logging to track the session token
		log.Printf("Session token: %s", r.Header.Get("Authorization"))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Query messages between users
	rows, err := database.DBInstance.DB.Query(`
        SELECT id, sender_id, receiver_id, content, created_at, is_read 
        FROM messages 
        WHERE (sender_id = ? AND receiver_id = ?) 
        OR (sender_id = ? AND receiver_id = ?)
        ORDER BY created_at DESC
        LIMIT ? OFFSET ?
    `, currentUserId, userId, userId, currentUserId, limit, offset)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []struct {
		ID         int    `json:"id"`
		SenderID   int    `json:"sender_id"`
		ReceiverID int    `json:"receiver_id"`
		Content    string `json:"content"`
		CreatedAt  string `json:"created_at"`
		IsRead     bool   `json:"is_read"`
	}

	for rows.Next() {
		var msg struct {
			ID         int    `json:"id"`
			SenderID   int    `json:"sender_id"`
			ReceiverID int    `json:"receiver_id"`
			Content    string `json:"content"`
			CreatedAt  string `json:"created_at"`
			IsRead     bool   `json:"is_read"`
		}
		err := rows.Scan(&msg.ID, &msg.SenderID, &msg.ReceiverID, &msg.Content, &msg.CreatedAt, &msg.IsRead)
		if err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	// Mark messages as read
	_, err = database.DBInstance.DB.Exec(`
        UPDATE messages 
        SET is_read = TRUE 
        WHERE receiver_id = ? AND sender_id = ? AND is_read = FALSE
    `, currentUserId, userId)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
