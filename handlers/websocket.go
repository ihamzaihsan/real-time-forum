package handlers

import (
	"RTF/database"
	"database/sql"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Message struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

// Add a connection write mutex for each client
type SafeConn struct {
    conn *websocket.Conn
    mu   sync.Mutex
}

// Update the clients map to use SafeConn
var clients = make(map[int]*SafeConn)
var clientsMutex sync.RWMutex

// HandleWebSocket handles the WebSocket connections
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get token from query parameter
	sessionToken := r.URL.Query().Get("token")
	if sessionToken == "" {
		log.Println("[ERROR] Missing session token")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ERROR] WebSocket upgrade failed: %v", err)
		return
	}

    safeConn := &SafeConn{
        conn: conn,
    }

	// Set the Authorization header with the token
	r.Header.Set("Authorization", sessionToken)

	userID := getUserIDFromSession(r)
	if userID == 0 {
		log.Println("[ERROR] Invalid session token or user not found")
		conn.Close()
		return
	}

	username := getUsernameFromSession(r)
	if username == "" {
		log.Println("[ERROR] Failed to retrieve username for user ID:", userID)
		conn.Close()
		return
	}

	clientsMutex.Lock()
	clients[userID] = safeConn
	clientsMutex.Unlock()


	// Update user status to online
	_, err = database.DBInstance.DB.Exec(
		"UPDATE users SET is_online = TRUE WHERE id = ?",
		userID,
	)

	// Start goroutine for broadcasting user list updates
	go broadcastActiveUsers()

	defer func() {
		clientsMutex.Lock()
		delete(clients, userID)
		clientsMutex.Unlock()
		
		// Update user status to offline in database
		_, err := database.DBInstance.DB.Exec(
			"UPDATE users SET is_online = FALSE WHERE id = ?",
			userID,
		)
		if err != nil {
			log.Printf("[ERROR] Failed to update offline status: %v", err)
		}
		
		// Broadcast updated user list to all clients
		go broadcastActiveUsers()
		
		conn.Close()
		
	}()

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		switch msg.Type {
		case "private_message":
			handlePrivateMessage(userID, username, msg.Content)
			case "typing_status":
				handleTypingStatus(userID, username, msg.Content)
		case "ping":
			err = safeConn.WriteJSON(Message{
				Type: "pong",
				Content: map[string]interface{}{
					"timestamp": time.Now().Format(time.RFC3339),
				},
			})
			if err != nil {
				log.Printf("[ERROR] Failed to send pong to user %d (%s): %v", userID, username, err)
			}
		default:
			log.Printf("[WARN] Unknown message type from user %d (%s): %s", userID, username, msg.Type)
		}
	}
}
func handleTypingStatus(senderID int, senderUsername string, content interface{}) {
    
    contentMap, ok := content.(map[string]interface{})
    if !ok {
        return
    }

    receiverID, ok := contentMap["receiver_id"].(float64)
    if !ok {
        return
    }

    isTyping, ok := contentMap["isTyping"].(bool)
    if !ok {
        return
    }

    
    clientsMutex.RLock()
    if recipientConn, ok := clients[int(receiverID)]; ok {
        err := recipientConn.WriteJSON(Message{
            Type: "typing_status",
            Content: map[string]interface{}{
                "user_id":  senderID,
                "username": senderUsername,
                "isTyping": isTyping,
            },
        })
        if err != nil {
            log.Printf("[DEBUG] Error sending typing status: %v", err)
        }
    } else {
        log.Printf("[DEBUG] Recipient connection not found")
    }
    clientsMutex.RUnlock()
}
// Create a safe write method
func (sc *SafeConn) WriteJSON(v interface{}) error {
    sc.mu.Lock()
    defer sc.mu.Unlock()
    return sc.conn.WriteJSON(v)
}

// Periodically sends the user list to each connected client.
func broadcastActiveUsers() {
    for {
        // Get current userID from the clients map
        clientsMutex.RLock()
        for userID := range clients {
            // Get active users excluding current user
            users, err := getActiveUsers(database.DBInstance.DB, userID)
            if err != nil {
                log.Printf("[ERROR] Failed to fetch active users: %v", err)
                continue
            }
            message := Message{
                Type:    "users_list",
                Content: users,
            }
            // Send the filtered list to this specific client
            if client, ok := clients[userID]; ok {
                err := client.WriteJSON(message)
                if err != nil {
                    log.Printf("[ERROR] Failed to broadcast user list: %v", err)
                }
            }
        }
        clientsMutex.RUnlock()
        time.Sleep(30 * time.Second)
    }
}
func handlePrivateMessage(senderID int, senderUsername string, content interface{}) {
	contentMap, ok := content.(map[string]interface{})
	if !ok {
		log.Printf("[WARN] Invalid private message format from user %d (%s)", senderID, senderUsername)
		return
	}

	receiverID, ok := contentMap["receiver_id"].(float64)
	if !ok {
		log.Printf("[WARN] Missing receiver_id in private message from user %d (%s)", senderID, senderUsername)
		return
	}

	messageContent, ok := contentMap["message"].(string)
	if !ok {
		log.Printf("[WARN] Missing or invalid message content from user %d (%s)", senderID, senderUsername)
		return
	}

	_, err := database.DBInstance.DB.Exec(
		"INSERT INTO messages (sender_id, receiver_id, content) VALUES (?, ?, ?)",
		senderID, int(receiverID), messageContent,
	)
	if err != nil {
		log.Printf("[ERROR] Failed to store private message from user %d to %d: %v", senderID, int(receiverID), err)
		return
	}

	clientsMutex.RLock()
	if recipientConn, ok := clients[int(receiverID)]; ok {
		err = recipientConn.WriteJSON(Message{
			Type: "private_message",
			Content: map[string]interface{}{
				"sender_id": senderID,
				"sender": senderUsername,
				"message": messageContent,
				"timestamp": time.Now().Format(time.RFC3339),
			},
		})
		
		if err != nil {
			log.Printf("[ERROR] Failed to deliver private message to user %d: %v", int(receiverID), err)
		}
	}
	clientsMutex.RUnlock()
}

func getUserIDFromSession(r *http.Request) int {
	sessionToken := r.Header.Get("Authorization")
	var userID int
	err := database.DBInstance.DB.QueryRow(
		"SELECT users.id FROM users JOIN sessions ON users.email = sessions.email WHERE sessions.session_token = ?",
		sessionToken,
	).Scan(&userID)
	if err != nil {
		log.Printf("[ERROR] Failed to retrieve user ID: %v", err)
		return 0
	}
	return userID
}

func getUsernameFromSession(r *http.Request) string {
	sessionToken := r.Header.Get("Authorization")
	var username string
	err := database.DBInstance.DB.QueryRow(
		"SELECT users.username FROM users JOIN sessions ON users.email = sessions.email WHERE sessions.session_token = ?",
		sessionToken,
	).Scan(&username)
	if err != nil {
		log.Printf("[ERROR] Failed to retrieve username: %v", err)
		return ""
	}
	return username
}

func getActiveUsers(db *sql.DB, currentUserID int) ([]map[string]interface{}, error) {

    rows, err := db.Query(`
        WITH MessageInfo AS (
            SELECT 
                u.id,
                u.username,
                u.is_online,
                MAX(CASE 
                    WHEN (m.sender_id = ? AND m.receiver_id = u.id) OR 
                        (m.receiver_id = ? AND m.sender_id = u.id)
                    THEN m.created_at 
                END) as last_message_time
            FROM users u
            LEFT JOIN messages m ON (u.id = m.sender_id OR u.id = m.receiver_id)
            WHERE u.id != ?
            GROUP BY u.id, u.username, u.is_online
        )
        SELECT * FROM MessageInfo
        ORDER BY 
            CASE WHEN last_message_time IS NOT NULL THEN 0 ELSE 1 END,
            last_message_time DESC NULLS LAST,
            username ASC
    `, currentUserID, currentUserID, currentUserID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var (
			id              int
			username        string
			isOnline       bool
			lastMessageTime sql.NullString
		)
		
		if err := rows.Scan(&id, &username, &isOnline, &lastMessageTime); err != nil {
			log.Printf("[WARN] Skipping user due to scan error: %v", err)
			continue
		}

		users = append(users, map[string]interface{}{
			"id":              id,
			"username":        username,
			"isOnline":        isOnline,
			"lastMessageTime": lastMessageTime.String,
		})
	}
	return users, nil
}