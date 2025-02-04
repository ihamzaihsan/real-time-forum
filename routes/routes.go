package routes

import (
	handlers "RTF/handlers"
	"net/http"
	"path/filepath"
)

func InitRoutes() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.ServeFile(w, r, filepath.Join("frontend", "index.html"))
	})
	http.HandleFunc("/register", handlers.ServeRegister)
	http.HandleFunc("/ws", handlers.HandleWebSocket)
	http.HandleFunc("/logout", handlers.ServeLogout)
	http.HandleFunc("/login", handlers.ServeLogin)
	http.HandleFunc("/messages/", handlers.ServeMessages)
	http.HandleFunc("/create_post", handlers.ServeCreatePost)
	http.HandleFunc("/posts", handlers.ServePosts)
	http.HandleFunc("/post/", handlers.ServePostByID)
	http.HandleFunc("/comment", handlers.ServeCreateComment)
	http.HandleFunc("/comments", handlers.ServeGetComments)
	http.HandleFunc("/like", handlers.ServeLike)
	http.HandleFunc("/comment/like", handlers.ServeCommentLike)

}
