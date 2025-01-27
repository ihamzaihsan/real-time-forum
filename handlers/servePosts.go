package handlers

import (
    "RTF/database"
    "RTF/models"
    "encoding/json"
    "net/http"
)

func ServePosts(w http.ResponseWriter, r *http.Request) {
    rows, err := database.DBInstance.DB.Query(`
        SELECT p.id, p.title, p.content, p.username, p.created_at, p.likes, p.dislikes, p.user_reaction, c.name
        FROM posts p
        LEFT JOIN post_categories pc ON p.id = pc.post_id
        LEFT JOIN categories c ON pc.category_id = c.id
    `)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    postsMap := make(map[int]*models.Post)
    for rows.Next() {
        var post models.Post
        var category models.Category
        if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.Username, &post.CreatedAt, &post.Likes, &post.Dislikes, &post.UserReaction, &category.Name); err != nil {
            http.Error(w, "Database error", http.StatusInternalServerError)
            return
        }
        if existingPost, exists := postsMap[post.ID]; exists {
            existingPost.Categories = append(existingPost.Categories, category.Name)
        } else {
            post.Categories = []string{category.Name}
            postsMap[post.ID] = &post
        }
    }

    var posts []models.Post
    for _, post := range postsMap {
        posts = append(posts, *post)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(posts)
}
