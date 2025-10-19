package main

import (
	"fmt"
	"foodshop/internal/auth"
	"foodshop/internal/database"
	"foodshop/internal/handler"
	"foodshop/internal/middleware"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	server         = "127.0.0.1"
	port           = "8080"
	db             *database.Sqlite
	tokenBlacklist *auth.TokenBlacklist
)

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
	jwtSecret := os.Getenv("JWTSECRET")
	if jwtSecret == "" {
		log.Fatal("Missing JWT Secret")
	}
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
	mux.HandleFunc("/", handler.IndexHandler())
	mux.HandleFunc("/registration", handler.RegistrationHandler(db))
	mux.HandleFunc("/login", handler.LoginHandler(db))
	mux.HandleFunc("/refresh", handler.RefreshHandler(db))

	// Protected endpoints (authentication required)
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("/logout", handler.LogoutHandler(tokenBlacklist))
	protectedMux.HandleFunc("/profile", handler.ProfileHandler(db))

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
