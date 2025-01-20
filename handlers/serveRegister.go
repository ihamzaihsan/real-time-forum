package handlers

import (
	models "RTF/models"
	database "RTF/database"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func ServeRegister(w http.ResponseWriter, r *http.Request) {
	var user models.User

	// Decode the incoming JSON data
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Validate input (you can extend this validation)
	if user.Username == "" || user.Email == "" || user.Password == "" {
		http.Error(w, "Username, email, and password are required", http.StatusBadRequest)
		return
	}

	// Register the user by calling the RegisterUser function
	if err := RegisterUser(&user); err != nil {
		http.Error(w, "Error registering user", http.StatusInternalServerError)
		return
	}

	// Send a success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

func RegisterUser(user *models.User) error {
	// Hash the password before storing it
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Set the password to the hashed version
	user.Password = string(hashedPassword)
	user.JoinDate = time.Now()

	// Query to insert the new user into the database
	query := `INSERT INTO users (username, email, password, first_name, last_name, age, gender, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = database.DBInstance.DB.Exec(query, user.Username, user.Email, user.Password, user.FirstName, user.LastName, user.Age, user.Gender, user.JoinDate)
	return err
}
