package models

type UserProfile struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Age       int    `json:"age"`
	Gender    string `json:"gender"`
}
