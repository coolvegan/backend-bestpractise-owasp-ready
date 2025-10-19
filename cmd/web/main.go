package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"foodshop/internal/database"
	"foodshop/internal/middleware"
	"foodshop/internal/models"
	"foodshop/internal/validator"
	"log"
	"net/http"
	"time"
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

	// Sanitize inputs
	reg.Username = validator.SanitizeInput(reg.Username)
	reg.Email = validator.SanitizeInput(reg.Email)

	// Validate username with enhanced security rules
	if err := validator.ValidateUsername(reg.Username); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: err.Error(),
		})
		return
	}

	// Validate password with strong requirements
	if err := validator.ValidatePassword(reg.Password); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: err.Error(),
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
	if err := validator.ValidateEmail(reg.Email); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: err.Error(),
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

	// Create rate limiter: 10 requests per second, burst of 20
	rateLimiter := middleware.NewRateLimiter(10, 20)

	// Create router/mux
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/", IndexHandler)
	mux.HandleFunc("/login", LoginHandler)
	mux.HandleFunc("/logout", IndexHandler)
	mux.HandleFunc("/registration", RegistrationHandler)

	// Build middleware chain (order matters!)
	var handler http.Handler = mux

	// Recovery must be first to catch panics from other middleware
	handler = middleware.Recovery(handler)

	// Logging
	handler = middleware.Logger(handler)

	// Security headers
	handler = middleware.SecurityHeaders(handler)

	// CORS - allow localhost for development
	handler = middleware.CORS([]string{"http://localhost:3000", "http://localhost:8080"})(handler)

	// Rate limiting
	handler = rateLimiter.Limit(handler)

	// Request size limit (1MB)
	handler = middleware.MaxBytesReader(1024 * 1024)(handler)

	// Request timeout (30 seconds)
	handler = middleware.Timeout(30 * time.Second)(handler)

	// Configure server with security best practices
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", server, port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on %s:%s with security middleware enabled", server, port)
	log.Printf("Security features: Rate limiting, CORS, Security headers, Request size limits, Timeouts")

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
