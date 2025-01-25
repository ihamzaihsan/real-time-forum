package routes

import (
	handlers "RTF/handlers"
	"net/http"
)

func InitRoutes() {
	http.HandleFunc("/", handlers.ServeMainPage)
	http.HandleFunc("/register", handlers.ServeRegister)
	http.HandleFunc("/ws", handlers.HandleWebSocket)
	http.HandleFunc("/logout", handlers.ServeLogout)
	http.HandleFunc("/login", handlers.ServeLogin)
}
