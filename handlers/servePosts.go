package handlers

import (
	"RTF/database"
	"RTF/models"
	"database/sql"
	"net/http"
	"strings"
)

const postSelect = `SELECT p.id,p.user_id,p.title,p.content,u.username,p.created_at,
 (SELECT COUNT(*) FROM likes WHERE post_id=p.id AND is_like=1) AS likes,
 (SELECT COUNT(*) FROM likes WHERE post_id=p.id AND is_like=0) AS dislikes,
 COALESCE((SELECT GROUP_CONCAT(c.name) FROM categories c JOIN post_categories pc ON pc.category_id=c.id WHERE pc.post_id=p.id),'')
 FROM posts p JOIN users u ON p.user_id=u.id `

func readPosts(rows *sql.Rows) ([]models.Post, error) {
	posts := make([]models.Post, 0)
	for rows.Next() {
		var p models.Post
		var categories string
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.Username, &p.CreatedAt, &p.Likes, &p.Dislikes, &categories); err != nil {
			return nil, err
		}
		if categories != "" {
			p.Categories = strings.Split(categories, ",")
		} else {
			p.Categories = []string{}
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

func ServePosts(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		http.Error(w, "Use limit 1–50 and a nonnegative offset", http.StatusBadRequest)
		return
	}
	query, category, sort := strings.TrimSpace(r.URL.Query().Get("q")), strings.TrimSpace(r.URL.Query().Get("category")), r.URL.Query().Get("sort")
	if len([]rune(query)) > 200 || len(category) > 32 {
		http.Error(w, "Search or category is too long", http.StatusBadRequest)
		return
	}
	order := "julianday(p.created_at) DESC,p.id DESC"
	if sort == "popular" {
		order = "likes DESC," + order
	} else if sort != "" && sort != "newest" {
		http.Error(w, "Sort must be newest or popular", http.StatusBadRequest)
		return
	}
	where := ` WHERE (?='' OR instr(lower(p.title),lower(?))>0 OR instr(lower(p.content),lower(?))>0 OR instr(lower(u.username),lower(?))>0)
 AND (?='' OR EXISTS(SELECT 1 FROM post_categories pc JOIN categories c ON c.id=pc.category_id WHERE pc.post_id=p.id AND c.name=?))`
	args := []interface{}{query, query, query, query, category, category}
	var total int
	if err := database.DBInstance.DB.QueryRow("SELECT COUNT(*) FROM posts p JOIN users u ON u.id=p.user_id"+where, args...).Scan(&total); err != nil {
		serverError(w, err)
		return
	}
	rows, err := database.DBInstance.DB.Query(postSelect+where+" ORDER BY "+order+" LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	posts, err := readPosts(rows)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"posts": posts, "total": total, "limit": limit, "offset": offset})
}
