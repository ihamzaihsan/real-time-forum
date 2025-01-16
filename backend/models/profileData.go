package models

type ProfileData struct {
    Username     string `json:"username"`
    Email        string `json:"email"`
    JoinDate     string `json:"join_date"`
    PostCount    int    `json:"post_count"`
    CommentCount int    `json:"comment_count"`
    IsLoggedIn   bool   `json:"is_logged_in"`
}
