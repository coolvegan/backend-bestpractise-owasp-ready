package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"foodshop/internal/auth"
	"foodshop/internal/database"
	"foodshop/internal/middleware"
	"foodshop/internal/models"
	"foodshop/internal/validator"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	server         = "127.0.0.1"
	port           = "8080"
	db             *database.Sqlite
	tokenBlacklist *auth.TokenBlacklist
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Welcome to Foodshop API",
		"version": "1.0.0",
	})
}

// ProfileHandler returns the authenticated user's profile (protected endpoint)
func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Get user info from context (added by AuthMiddleware)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Unauthorized",
		})
		return
	}

	// Fetch user from database
	user, err := db.GetUserByID(userID)
	if err != nil {
		log.Printf("Error fetching user profile: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Failed to fetch profile",
		})
		return
	}

	// Return user profile
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"is_active":  user.IsActive,
			"created_at": user.CreatedAt,
		},
	})
}

// LoginHandler handles user login and returns JWT token
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Parse request body
	var loginReq models.UserLogin
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Invalid request data",
		})
		return
	}

	// Sanitize inputs
	loginReq.Username = validator.SanitizeInput(loginReq.Username)

	// Validate inputs
	if loginReq.Username == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Username is required",
		})
		return
	}

	if loginReq.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Password is required",
		})
		return
	}

	// Verify credentials
	user, err := db.VerifyPassword(loginReq.Username, loginReq.Password)
	if err != nil {
		// Use generic error message to prevent username enumeration
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Invalid username or password",
		})
		return
	}

	// Generate JWT token
	token, err := auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Failed to generate authentication token",
		})
		return
	}

	// Generate refresh token
	refreshToken, err := auth.GenerateRefreshToken(user.ID, user.Username)
	if err != nil {
		log.Printf("Error generating refresh token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Failed to generate refresh token",
		})
		return
	}

	// Success response
	response := models.LoginResponse{
		Message:      "Login successful",
		Token:        token,
		RefreshToken: refreshToken,
	}
	response.User.ID = user.ID
	response.User.Username = user.Username
	response.User.Email = user.Email

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// LogoutHandler handles user logout by blacklisting the token
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Missing authorization header",
		})
		return
	}

	// Parse Bearer token
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Invalid authorization header format",
		})
		return
	}

	tokenString := parts[1]

	// Validate token to get expiration time
	claims, err := auth.ValidateToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrUserLogin{
			Message: "Invalid or expired token",
		})
		return
	}

	// Add token to blacklist
	tokenBlacklist.Add(tokenString, claims.ExpiresAt.Time)

	// Success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logout successful",
	})
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

	// Initialize JWT secret (in production, load from environment variable)
	jwtSecret := "your-super-secret-jwt-key-change-in-production"
	auth.SetJWTSecret(jwtSecret)
	log.Printf("JWT authentication enabled")

	// Initialize token blacklist
	tokenBlacklist = auth.NewTokenBlacklist()
	log.Printf("Token blacklist initialized")

	// Create rate limiter: 10 requests per second, burst of 20
	rateLimiter := middleware.NewRateLimiter(10, 20)

	// Create router/mux
	mux := http.NewServeMux()

	// Public endpoints (no authentication required)
	mux.HandleFunc("/", IndexHandler)
	mux.HandleFunc("/registration", RegistrationHandler)
	mux.HandleFunc("/login", LoginHandler)

	// Protected endpoints (authentication required)
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("/logout", LogoutHandler)
	protectedMux.HandleFunc("/profile", ProfileHandler)

	// Apply auth middleware to protected routes
	authMiddleware := middleware.AuthMiddleware(tokenBlacklist)
	mux.Handle("/logout", authMiddleware(protectedMux))
	mux.Handle("/profile", authMiddleware(protectedMux))

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
