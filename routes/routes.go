package routes

import (
	handlers "RTF/handlers"
	"net/http"
	"path/filepath"
)

func Register(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		switch r.URL.Path {
		case "/", "/chat", "/profile", "/moderation":
		default:
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join("frontend", "index.html"))
	})

	mux.HandleFunc("/register", handlers.ServeRegister)
	mux.HandleFunc("/ws", handlers.HandleWebSocket)
	mux.HandleFunc("/logout", handlers.ServeLogout)
	mux.HandleFunc("/login", handlers.ServeLogin)
	mux.HandleFunc("/messages/", handlers.ServeMessages)
	mux.HandleFunc("/create_post", handlers.ServeCreatePost)
	mux.HandleFunc("/posts", handlers.ServePosts)
	mux.HandleFunc("/post/", handlers.ServePostByID)
	mux.HandleFunc("/comment", handlers.ServeCreateComment)
	mux.HandleFunc("/comments", handlers.ServeGetComments)
	mux.HandleFunc("/like", handlers.ServeLike)
	mux.HandleFunc("/comment/like", handlers.ServeCommentLike)
	mux.HandleFunc("/categories", handlers.ServeCategories)
	mux.HandleFunc("/profile/", handlers.ServeProfile)
	mux.HandleFunc("/check-auth", handlers.CheckAuth)
	mux.HandleFunc("/delete-post", handlers.ServeDeletePost)
	mux.HandleFunc("/reports", handlers.ServeReports)
	mux.HandleFunc("/moderate", handlers.ServeModerate)

}
