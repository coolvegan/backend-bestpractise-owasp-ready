package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"foodshop/internal/database"
	"foodshop/internal/models"
	"log"
	"net/http"
	"strings"
)

var (
	server = "127.0.0.1"
	port   = "8080"
	db     *database.Sqlite
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		return
	}

}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}
	var ul models.UserLogin
	if err := json.NewDecoder(r.Body).Decode(&ul); err != nil {
		json.NewEncoder(w).Encode(models.ErrUserLogin{Message: "The provided data is wrong!"})
	}
	if ul.Username == "" {
		json.NewEncoder(w).Encode(models.ErrUserLogin{Message: "The provided data has missing pieces!"})
	}

}

// RegistrationHandler handles user registration requests.
func RegistrationHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Parse request body
	var reg models.UserRegistration
	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Invalid request data",
		})
		return
	}

	// Validate username
	reg.Username = strings.TrimSpace(reg.Username)
	if reg.Username == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Username is required",
		})
		return
	}

	// Validate username length and format
	if len(reg.Username) < 3 || len(reg.Username) > 50 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Username must be between 3 and 50 characters",
		})
		return
	}

	// Validate password
	if reg.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Password is required",
		})
		return
	}

	// Validate password strength
	if len(reg.Password) < 8 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Password must be at least 8 characters long",
		})
		return
	}

	// Validate password verification
	if reg.PasswordVerification == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Password verification is required",
		})
		return
	}

	// Check if passwords match
	if reg.Password != reg.PasswordVerification {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Passwords do not match",
		})
		return
	}

	// Validate email if provided
	reg.Email = strings.TrimSpace(reg.Email)
	if reg.Email != "" && !strings.Contains(reg.Email, "@") {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Invalid email format",
		})
		return
	}

	// Create user in database
	user, err := db.CreateUser(reg.Username, reg.Password, reg.Email)
	if err != nil {
		if errors.Is(err, database.ErrUserExists) {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Username already exists",
			})
			return
		}

		log.Printf("Error creating user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Failed to create user",
		})
		return
	}

	// Success response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User created successfully",
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

func main() {
	// Initialize database
	var err error
	repo, err := database.New("./data/foodshop.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer repo.Close()

	// Type assertion to access Sqlite-specific methods
	var ok bool
	db, ok = repo.(*database.Sqlite)
	if !ok {
		log.Fatal("Failed to cast repository to Sqlite")
	}

	// Initialize schema
	if err := db.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	log.Printf("Database initialized successfully")

	// Register handlers
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/logout", IndexHandler)
	http.HandleFunc("/registration", RegistrationHandler)

	log.Printf("Server starting on %s:%s", server, port)
	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", server, port), nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
