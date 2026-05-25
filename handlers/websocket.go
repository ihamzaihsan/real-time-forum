package handlers

import (
	"RTF/database"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

var upgrader = websocket.Upgrader{CheckOrigin: sameOrigin, HandshakeTimeout: 5 * time.Second}

type Message struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}
type SafeConn struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	userID int
	token  string
}

var clients = make(map[int]map[*SafeConn]struct{})
var clientsMutex sync.RWMutex
var connectionsWG sync.WaitGroup
var socketLimiter = newRateLimiter(120, time.Minute)

func CloseConnections() {
	for _, client := range connectionSnapshot(-1) {
		client.conn.Close()
	}
	connectionsWG.Wait()
}

func (sc *SafeConn) WriteJSON(v interface{}) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if err := sc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	return sc.conn.WriteJSON(v)
}
func (sc *SafeConn) ping() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second))
}
func connectionSnapshot(userID int) []*SafeConn {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()
	result := []*SafeConn{}
	for id, connections := range clients {
		if userID >= 0 && id != userID {
			continue
		}
		for client := range connections {
			result = append(result, client)
		}
	}
	return result
}
func broadcastMessage(message Message) {
	for _, client := range connectionSnapshot(-1) {
		if err := client.WriteJSON(message); err != nil {
			client.conn.Close()
		}
	}
}
func sendToUser(userID int, message Message) {
	for _, client := range connectionSnapshot(userID) {
		if _, err := lookupSession(client.token); err != nil {
			client.conn.Close()
			continue
		}
		if err := client.WriteJSON(message); err != nil {
			client.conn.Close()
		}
	}
}
func closeSessionConnections(token string) {
	for _, client := range connectionSnapshot(-1) {
		if client.token == token {
			client.conn.Close()
		}
	}
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	var s session
	token := requestToken(r)
	if token != "" {
		var err error
		s, err = lookupSession(token)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Session expired", http.StatusUnauthorized)
			} else {
				serverError(w, err)
			}
			return
		}
	}
	if !sameOrigin(r) {
		http.Error(w, "Cross-origin connection rejected", http.StatusForbidden)
		return
	}
	// Reserve a slot before upgrading to bound simultaneous connections.
	client := &SafeConn{userID: s.UserID, token: token}
	clientsMutex.Lock()
	count := 0
	for _, group := range clients {
		count += len(group)
	}
	if count >= 1000 || (s.UserID > 0 && len(clients[s.UserID]) >= 8) {
		clientsMutex.Unlock()
		http.Error(w, "Connection limit reached", http.StatusTooManyRequests)
		return
	}
	if clients[s.UserID] == nil {
		clients[s.UserID] = make(map[*SafeConn]struct{})
	}
	// Upgrade under this short lock so snapshots never see an uninitialized connection.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if len(clients[s.UserID]) == 0 {
			delete(clients, s.UserID)
		}
		clientsMutex.Unlock()
		return
	}
	client.conn = conn
	clients[s.UserID][client] = struct{}{}
	connectionsWG.Add(1)
	defer connectionsWG.Done()
	clientsMutex.Unlock()
	if s.UserID > 0 {
		broadcastActiveUsers()
	}
	done := make(chan struct{})
	defer func() {
		close(done)
		conn.Close()
		clientsMutex.Lock()
		delete(clients[s.UserID], client)
		if len(clients[s.UserID]) == 0 {
			delete(clients, s.UserID)
		}
		clientsMutex.Unlock()
		if s.UserID > 0 {
			broadcastActiveUsers()
		}
	}()
	conn.SetReadLimit(8 * 1024)
	conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(70 * time.Second)) })
	go func() {
		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()
		for {
			select {
			case <-done:
				return
			case <-heartbeat.C:
				if token != "" {
					if _, err := lookupSession(token); err != nil {
						conn.Close()
						return
					}
				}
				if err := client.ping(); err != nil {
					conn.Close()
					return
				}
			}
		}
	}()
	for {
		var input struct {
			Type    string          `json:"type"`
			Content json.RawMessage `json:"content"`
		}
		if err := conn.ReadJSON(&input); err != nil {
			return
		}
		if s.UserID == 0 {
			if input.Type == "ping" {
				client.WriteJSON(Message{Type: "pong", Content: nil})
			} else {
				client.WriteJSON(Message{Type: "error", Content: map[string]string{"message": "Sign in to send messages"}})
			}
			continue
		}
		if _, err := lookupSession(token); err != nil {
			return
		}
		if !socketLimiter.allow(fmt.Sprint(s.UserID)) {
			client.WriteJSON(Message{Type: "error", Content: map[string]string{"message": "Too many messaging events. Try again later."}})
			continue
		}
		switch input.Type {
		case "private_message":
			handlePrivateMessage(client, s, input.Content)
		case "typing_status":
			handleTypingStatus(s, input.Content)
		case "ping":
			client.WriteJSON(Message{Type: "pong", Content: nil})
		default:
			client.WriteJSON(Message{Type: "error", Content: map[string]string{"message": "Unknown message type"}})
		}
	}
}

func decodePayload(raw json.RawMessage, value interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(value)
}
func handlePrivateMessage(client *SafeConn, s session, raw json.RawMessage) {
	var input struct {
		ReceiverID int    `json:"receiver_id"`
		Text       string `json:"message"`
		ClientID   string `json:"client_id"`
	}
	err := decodePayload(raw, &input)
	input.Text = strings.TrimSpace(input.Text)
	fail := func(message string) {
		client.WriteJSON(Message{Type: "message_error", Content: map[string]string{"client_id": input.ClientID, "message": message}})
	}
	if err != nil || input.ReceiverID < 1 || input.ReceiverID == s.UserID || input.Text == "" || len([]rune(input.Text)) > 2000 || input.ClientID == "" || len(input.ClientID) > 64 {
		fail("Choose another user and send 1–2,000 characters with a client ID")
		return
	}
	// The client ID makes a repeated send idempotent if an acknowledgment was lost.
	m := messageRecord{SenderID: s.UserID, ReceiverID: input.ReceiverID, Content: input.Text, CreatedAt: time.Now().UTC(), Sender: s.Username, ClientID: input.ClientID}
	tx, err := database.DBInstance.DB.Begin()
	if err != nil {
		fail("Could not save message")
		return
	}
	defer tx.Rollback()
	var receiver int
	err = tx.QueryRow("SELECT id FROM users WHERE id=?", input.ReceiverID).Scan(&receiver)
	if errors.Is(err, sql.ErrNoRows) {
		fail("Recipient not found")
		return
	}
	if err != nil {
		log.Printf("message recipient: %v", err)
		fail("Could not save message")
		return
	}
	result, err := tx.Exec("INSERT INTO messages(sender_id,receiver_id,content,created_at,client_id) VALUES(?,?,?,?,?) ON CONFLICT(sender_id,client_id) WHERE client_id IS NOT NULL DO NOTHING", s.UserID, input.ReceiverID, input.Text, m.CreatedAt, input.ClientID)
	if err != nil {
		log.Printf("save message: %v", err)
		fail("Could not save message")
		return
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		fail("Could not save message")
		return
	}
	err = tx.QueryRow("SELECT id,receiver_id,content,created_at FROM messages WHERE sender_id=? AND client_id=?", s.UserID, input.ClientID).Scan(&m.ID, &m.ReceiverID, &m.Content, &m.CreatedAt)
	if err != nil {
		fail("Could not confirm message")
		return
	}
	if m.ReceiverID != input.ReceiverID || m.Content != input.Text {
		fail("Client ID was already used for another message")
		return
	}
	if err := tx.Commit(); err != nil {
		fail("Could not save message")
		return
	}
	sendToUser(s.UserID, Message{Type: "message_sent", Content: m})
	if inserted > 0 {
		sendToUser(m.ReceiverID, Message{Type: "private_message", Content: m})
		broadcastActiveUsers()
	}
}
func handleTypingStatus(s session, raw json.RawMessage) {
	var input struct {
		ReceiverID int  `json:"receiver_id"`
		IsTyping   bool `json:"isTyping"`
	}
	if decodePayload(raw, &input) != nil || input.ReceiverID < 1 || input.ReceiverID == s.UserID {
		return
	}
	sendToUser(input.ReceiverID, Message{Type: "typing_status", Content: map[string]interface{}{"user_id": s.UserID, "username": s.Username, "isTyping": input.IsTyping}})
}

func broadcastActiveUsers() {
	snapshot := connectionSnapshot(-1)
	lists := make(map[int][]map[string]interface{})
	for _, client := range snapshot {
		if client.userID == 0 {
			continue
		}
		if _, err := lookupSession(client.token); err != nil {
			client.conn.Close()
			continue
		}
		users, exists := lists[client.userID]
		if !exists {
			var err error
			users, err = getActiveUsers(database.DBInstance.DB, client.userID)
			if err != nil {
				log.Printf("presence: %v", err)
				continue
			}
			lists[client.userID] = users
		}
		if err := client.WriteJSON(Message{Type: "users_list", Content: users}); err != nil {
			client.conn.Close()
		}
	}
}
func getActiveUsers(db *sql.DB, currentUserID int) ([]map[string]interface{}, error) {
	rows, err := db.Query(`SELECT u.id,u.username,
 (SELECT MAX(m.id) FROM messages m WHERE (m.sender_id=? AND m.receiver_id=u.id) OR (m.receiver_id=? AND m.sender_id=u.id)) AS last_id,
 (SELECT strftime('%Y-%m-%dT%H:%M:%fZ',m.created_at) FROM messages m WHERE (m.sender_id=? AND m.receiver_id=u.id) OR (m.receiver_id=? AND m.sender_id=u.id) ORDER BY m.id DESC LIMIT 1)
 FROM users u WHERE u.id!=? ORDER BY last_id DESC,u.username COLLATE NOCASE`, currentUserID, currentUserID, currentUserID, currentUserID, currentUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []map[string]interface{}{}
	clientsMutex.RLock()
	online := make(map[int]bool, len(clients))
	for id, group := range clients {
		online[id] = len(group) > 0
	}
	clientsMutex.RUnlock()
	for rows.Next() {
		var id int
		var username string
		var lastID sql.NullInt64
		var lastTime sql.NullString
		if err := rows.Scan(&id, &username, &lastID, &lastTime); err != nil {
			return nil, err
		}
		timeString := lastTime.String
		users = append(users, map[string]interface{}{"id": id, "username": username, "isOnline": online[id], "lastMessageTime": timeString})
	}
	return users, rows.Err()
}
