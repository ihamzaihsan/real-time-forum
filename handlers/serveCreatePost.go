package handlers

import (
    "RTF/database"
    "RTF/models"
    "encoding/json"
    "net/http"
    "time"
)

func ServeCreatePost(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var post models.Post
    if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
        http.Error(w, "Invalid input", http.StatusBadRequest)
        return
    }

    post.CreatedAt = time.Now()

    result, err := database.DBInstance.DB.Exec(
        "INSERT INTO posts (title, content, username, created_at, likes, dislikes, user_reaction) VALUES (?, ?, ?, ?, ?, ?, ?)",
        post.Title, post.Content, post.Username, post.CreatedAt, post.Likes, post.Dislikes, post.UserReaction,
    )
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    postID, err := result.LastInsertId()
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    for _, category := range post.Categories {
        _, err := database.DBInstance.DB.Exec(
            "INSERT INTO post_categories (post_id, category) VALUES (?, ?)",
            postID, category,
        )
        if err != nil {
            http.Error(w, "Database error", http.StatusInternalServerError)
            return
        }
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Post created successfully",
        "post_id": postID,
    })
}
