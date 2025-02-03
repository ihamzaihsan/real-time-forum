package handlers

import (
    "RTF/database"
    "RTF/models"
    "encoding/json"
    "net/http"
    "time"
)

func ServeCreateComment(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var comment models.Comment
    if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
        http.Error(w, "Invalid input", http.StatusBadRequest)
        return
    }

    // Get user ID from session
    userID := getUserIDFromSession(r)
    if userID == 0 {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    comment.UserID = userID
    comment.CreatedAt = time.Now()

    // Insert comment into database
    result, err := database.DBInstance.DB.Exec(
        "INSERT INTO comments (content, user_id, post_id, created_at) VALUES (?, ?, ?, ?)",
        comment.Content, comment.UserID, comment.PostID, comment.CreatedAt,
    )
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    commentID, _ := result.LastInsertId()
    comment.ID = int(commentID)

    // Broadcast new comment to WebSocket clients
    broadcastComment(comment)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(comment)
}

func ServeGetComments(w http.ResponseWriter, r *http.Request) {

    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    postID := r.URL.Query().Get("post_id")
    if postID == "" {
        http.Error(w, "Post ID required", http.StatusBadRequest)
        return
    }

    rows, err := database.DBInstance.DB.Query(`
        SELECT c.id, c.content, c.user_id, u.username, c.created_at,
        (SELECT COUNT(*) FROM likes WHERE comment_id = c.id AND is_like = 1) as likes,
        (SELECT COUNT(*) FROM likes WHERE comment_id = c.id AND is_like = 0) as dislikes
        FROM comments c
        JOIN users u ON c.user_id = u.id
        WHERE c.post_id = ?
        ORDER BY c.created_at DESC
    `, postID)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var comments []models.Comment
    for rows.Next() {
        var comment models.Comment
        err := rows.Scan(
            &comment.ID, &comment.Content, &comment.UserID, 
            &comment.Username, &comment.CreatedAt, 
            &comment.Likes, &comment.Dislikes,
        )
        if err != nil {
            http.Error(w, "Database error", http.StatusInternalServerError)
            return
        }
        comments = append(comments, comment)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(comments)
}

func broadcastComment(comment models.Comment) {
    message := Message{
        Type: "new_comment",
        Content: comment,
    }
    
    // Broadcast to all connected clients
    clientsMutex.RLock()
    for _, client := range clients {
        client.mu.Lock()
        client.conn.WriteJSON(message)
        client.mu.Unlock()
    }
    clientsMutex.RUnlock()
}
