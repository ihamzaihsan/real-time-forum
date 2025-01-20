package models

import "time"

type User struct {
    ID        int       `json:"id"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    Password  string    `json:"-"`
    FirstName string    `json:"first_name"`
    LastName  string    `json:"last_name"`
    Age       int       `json:"age"`
    Gender    string    `json:"gender"`
    JoinDate  time.Time `json:"created_at"`
}