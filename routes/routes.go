package routes

import (
    "net/http"
    handlers "RTF/handlers"
)

func InitRoutes() {
    http.HandleFunc("/", handlers.ServeMainPage)
    http.HandleFunc("/register", handlers.ServeRegister)
    http.HandleFunc("/ws", handlers.HandleWebSocket)
}
