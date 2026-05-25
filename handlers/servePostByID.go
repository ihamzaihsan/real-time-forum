package handlers

import (
	"RTF/database"
	"net/http"
	"strings"
)

func ServePostByID(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	id, err := positiveID(strings.TrimPrefix(r.URL.Path, "/post/"))
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}
	w.Header().Set("Vary", "Accept")
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		serveShell(w, r)
		return
	}
	rows, err := database.DBInstance.DB.Query(postSelect+" WHERE p.id=?", id)
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
	if len(posts) == 0 {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, posts[0])
}
