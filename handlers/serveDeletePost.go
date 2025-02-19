package handlers

import (
    "RTF/database"
    "net/http"
)

func ServeDeletePost(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    userID := getUserIDFromSession(r)
    if userID == 0 {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    postID := r.FormValue("post_id")
    if postID == "" {
        http.Error(w, "Invalid post ID", http.StatusBadRequest)
        return
    }

    // Verify post ownership
    var postOwnerID int
    err := database.DBInstance.DB.QueryRow(
        "SELECT user_id FROM posts WHERE id = ?", postID).Scan(&postOwnerID)
    if err != nil {
        http.Error(w, "Post not found", http.StatusNotFound)
        return
    }

    if postOwnerID != userID {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }

    // Delete the post
    _, err = database.DBInstance.DB.Exec("DELETE FROM posts WHERE id = ?", postID)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}
