package models

type message struct {
    ID        int       `json:"id"`
    SenderID  int       `json:"sender_id"`
    ReceiverID int      `json:"receiver_id"`
    Content   string    `json:"content"`
    CreatedAt time.Time `json:"created_at"`
    IsRead    bool      `json:"is_read"`
}