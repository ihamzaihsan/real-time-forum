package handlers

import (
	database "RTF/database"
	models "RTF/models"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func ServeRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		http.ServeFile(w, r, filepath.Join("frontend", "index.html"))
		return
	}
	var user models.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if user.Username == "" || user.Email == "" || user.Password == "" {
		http.Error(w, "Username, email, and password are required", http.StatusBadRequest)
		log.Println(user.Password)
		return
	}

	// passwordPattern := `^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]{8,}$`
	// passwordRegex := regexp.MustCompile(passwordPattern)

	// if !passwordRegex.MatchString(user.Password) {
	// 	http.Error(w, "Password must be at least 8 characters long and contain uppercase, lowercase, number and special character", http.StatusBadRequest)
	// 	return
	// }

	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	emailRegex := regexp.MustCompile(emailPattern)

	if !emailRegex.MatchString(user.Email) {
		http.Error(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	var count int
	err := database.DBInstance.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", user.Username).Scan(&count)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	err = database.DBInstance.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", user.Email).Scan(&count)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	if user.Age <= 0 {
		http.Error(w, "Age must be a positive number", http.StatusBadRequest)
		return
	}
	if user.Age > 120 {
		http.Error(w, "Age must be less than 120", http.StatusBadRequest)
		return
	}

	if err := RegisterUser(&user); err != nil {
		http.Error(w, "Error registering user", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	var userId int
	err = database.DBInstance.DB.QueryRow("SELECT id FROM users WHERE email = ?", user.Email).Scan(&userId)
	if err != nil {
		http.Error(w, "Error getting user ID", http.StatusInternalServerError)
		return
	}

	sessionToken := uuid.New().String()

	_, err = database.DBInstance.DB.Exec(`
				INSERT INTO sessions (session_token, email, expires_at) 
				VALUES (?, ?, ?)`,
		sessionToken, user.Email, time.Now().Add(24*time.Hour))

	if err != nil {
		http.Error(w, "Session creation failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})


	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User registered successfully",
		"token":   sessionToken,
		"user_id": userId,
	})
}

func RegisterUser(user *models.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	user.JoinDate = time.Now()

	query := `INSERT INTO users (username, email, password, first_name, last_name, age, gender, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = database.DBInstance.DB.Exec(query, user.Username, user.Email, user.Password, user.FirstName, user.LastName, user.Age, user.Gender, user.JoinDate)
	return err
}
