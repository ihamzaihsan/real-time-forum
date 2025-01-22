package handlers

import (
    "log"
    "net/http"

    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow all connections (for development purposes)
    },
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("WebSocket upgrade error:", err)
        return
    }
    defer conn.Close()

    for {
        // Read message from the client
        messageType, message, err := conn.ReadMessage()
        if err != nil {
            log.Println("WebSocket read error:", err)
            break
        }

        // Log the received message
        log.Printf("Received: %s", message)

        // Echo the message back to the client (or process it)
        if err := conn.WriteMessage(messageType, message); err != nil {
            log.Println("WebSocket write error:", err)
            break
        }
    }
}