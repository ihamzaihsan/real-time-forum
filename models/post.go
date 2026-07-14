package models

import "time"

type Post struct {
	Status     string    `json:"status"`
	ImagePath  string    `json:"image_path"`
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Username   string    `json:"username"`
	CreatedAt  time.Time `json:"created_at"`
	Likes      int       `json:"likes"`
	Dislikes   int       `json:"dislikes"`
	Categories []string  `json:"categories"`
}
