package handlers

import (
    "RTF/database"
    "encoding/json"
    "net/http"
    "strconv"
)

func ServeLike(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    sessionToken := r.Header.Get("Authorization")
    if sessionToken == "" {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }

    // Get user ID from session
    var userID int
    err := database.DBInstance.DB.QueryRow(
        "SELECT u.id FROM users u JOIN sessions s ON u.email = s.email WHERE s.session_token = ?",
        sessionToken,
    ).Scan(&userID)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Parse post ID and like status
    postID, err := strconv.Atoi(r.FormValue("post_id"))
    if err != nil {
        http.Error(w, "Invalid post ID", http.StatusBadRequest)
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
        "SELECT is_like FROM likes WHERE user_id = ? AND post_id = ?",
        userID, postID,
    ).Scan(&existingIsLike)

    if err == nil {
        // Update existing like/dislike
        _, err = database.DBInstance.DB.Exec(
            "UPDATE likes SET is_like = ? WHERE user_id = ? AND post_id = ?",
            isLike, userID, postID,
        )
    } else {
        // Insert new like/dislike
        _, err = database.DBInstance.DB.Exec(
            "INSERT INTO likes (user_id, post_id, is_like) VALUES (?, ?, ?)",
            userID, postID, isLike,
        )
    }

    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    // Get updated counts
    var likesCount, dislikesCount int
    err = database.DBInstance.DB.QueryRow(
        "SELECT COUNT(*) FROM likes WHERE post_id = ? AND is_like = true", postID,
    ).Scan(&likesCount)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    err = database.DBInstance.DB.QueryRow(
        "SELECT COUNT(*) FROM likes WHERE post_id = ? AND is_like = false", postID,
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
