package handlers

import (
    "RTF/database"
    "encoding/json"
    "net/http"
    "strconv"
)

func ServeCommentLike(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    sessionToken := r.Header.Get("Authorization")
    if sessionToken == "" {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }

    var userID int
    err := database.DBInstance.DB.QueryRow(
        "SELECT u.id FROM users u JOIN sessions s ON u.email = s.email WHERE s.session_token = ?",
        sessionToken,
    ).Scan(&userID)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Parse comment ID and like status
    commentID, err := strconv.Atoi(r.FormValue("comment_id"))
    if err != nil {
        http.Error(w, "Invalid comment ID", http.StatusBadRequest)
        return
    }

    isLike, err := strconv.ParseBool(r.FormValue("is_like"))
    if err != nil {
        http.Error(w, "Invalid like value", http.StatusBadRequest)
        return
    }

    // Check if user already liked/disliked
    var existingIsLike bool
    err = database.DBInstance.DB.QueryRow(
        "SELECT is_like FROM likes WHERE user_id = ? AND comment_id = ?",
        userID, commentID,
    ).Scan(&existingIsLike)

    if err == nil {
        // Update existing like/dislike
        _, err = database.DBInstance.DB.Exec(
            "UPDATE likes SET is_like = ? WHERE user_id = ? AND comment_id = ?",
            isLike, userID, commentID,
        )
    } else {
        // Insert new like/dislike
        _, err = database.DBInstance.DB.Exec(
            "INSERT INTO likes (user_id, comment_id, is_like) VALUES (?, ?, ?)",
            userID, commentID, isLike,
        )
    }

    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    // Get updated counts
    var likesCount, dislikesCount int
    err = database.DBInstance.DB.QueryRow(
        "SELECT COUNT(*) FROM likes WHERE comment_id = ? AND is_like = true", commentID,
    ).Scan(&likesCount)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    err = database.DBInstance.DB.QueryRow(
        "SELECT COUNT(*) FROM likes WHERE comment_id = ? AND is_like = false", commentID,
    ).Scan(&dislikesCount)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    response := struct {
        Likes    int `json:"likes"`
        Dislikes int `json:"dislikes"`
    }{
        Likes:    likesCount,
        Dislikes: dislikesCount,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
