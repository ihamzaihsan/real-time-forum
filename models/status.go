package models

import "time"

type UserStatus struct {
	Username string    `json:"username"`
	Status   bool      `json:"status"`
	LastSeen time.Time `json:"last_seen"`
}
