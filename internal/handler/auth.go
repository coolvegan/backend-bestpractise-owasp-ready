package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"foodshop/internal/auth"
	"foodshop/internal/database"
	"foodshop/internal/models"
	"foodshop/internal/validator"
	"net/http"
	"strings"
	"time"
)

// LoginHandler handles user login and returns JWT token
func LoginHandler(db *database.Sqlite) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var loginReq struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Invalid request data",
			})
			return
		}
		loginReq.Username = validator.SanitizeInput(loginReq.Username)
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
		isLocked, lockedUntil, err := db.IsAccountLocked(loginReq.Username)
		if err != nil && !errors.Is(err, database.ErrUserNotFound) {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Internal server error",
			})
			return
		}
		if isLocked {
			remainingTime := time.Until(lockedUntil)
			minutes := int(remainingTime.Minutes())
			w.WriteHeader(http.StatusLocked)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: fmt.Sprintf("Account locked due to too many failed login attempts. Try again in %d minutes.", minutes),
			})
			return
		}
		user, err := db.VerifyPassword(loginReq.Username, loginReq.Password)
		if err != nil {
			if !errors.Is(err, database.ErrUserNotFound) {
				db.IncrementFailedAttempts(loginReq.Username)
				attempts, _ := db.GetFailedAttempts(loginReq.Username)
				if attempts >= database.MaxLoginAttempts {
					db.LockAccount(loginReq.Username, database.LockoutDuration)
					w.WriteHeader(http.StatusLocked)
					json.NewEncoder(w).Encode(models.ErrUserLogin{
						Message: fmt.Sprintf("Account locked due to too many failed login attempts. Try again in %d minutes.", int(database.LockoutDuration.Minutes())),
					})
					return
				}
				remainingAttempts := database.MaxLoginAttempts - attempts
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(models.ErrUserLogin{
					Message: fmt.Sprintf("Invalid username or password. %d attempts remaining.", remainingAttempts),
				})
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Invalid username or password",
			})
			return
		}
		db.ResetFailedAttempts(loginReq.Username)
		token, err := auth.GenerateToken(user.ID, user.Username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Failed to generate authentication token",
			})
			return
		}
		refreshToken, err := auth.GenerateRefreshToken(user.ID, user.Username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Failed to generate refresh token",
			})
			return
		}
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
}

// LogoutHandler handles user logout by blacklisting the token
func LogoutHandler(tokenBlacklist *auth.TokenBlacklist) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Missing authorization header",
			})
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Invalid authorization header format",
			})
			return
		}
		tokenString := parts[1]
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Invalid or expired token",
			})
			return
		}
		tokenBlacklist.Add(tokenString, claims.ExpiresAt.Time)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Logout successful",
		})
	}
}

// RegistrationHandler handles user registration requests.
func RegistrationHandler(db *database.Sqlite) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var reg struct {
			Username             string `json:"username"`
			Password             string `json:"password"`
			PasswordVerification string `json:"password_verification"`
			Email                string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Invalid request data",
			})
			return
		}
		reg.Username = validator.SanitizeInput(reg.Username)
		reg.Email = validator.SanitizeInput(reg.Email)
		if err := validator.ValidateUsername(reg.Username); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: err.Error(),
			})
			return
		}
		if err := validator.ValidatePassword(reg.Password); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: err.Error(),
			})
			return
		}
		if reg.PasswordVerification == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Password verification is required",
			})
			return
		}
		if reg.Password != reg.PasswordVerification {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Passwords do not match",
			})
			return
		}
		if err := validator.ValidateEmail(reg.Email); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: err.Error(),
			})
			return
		}
		user, err := db.CreateUser(reg.Username, reg.Password, reg.Email)
		if err != nil {
			if errors.Is(err, database.ErrUserExists) {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(models.ErrUserLogin{
					Message: "Username already exists",
				})
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Failed to create user",
			})
			return
		}
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
}

func RefreshHandler(db *database.Sqlite) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Missing or invalid refresh_token",
			})
			return
		}
		claims, err := auth.ValidateToken(req.RefreshToken)
		if err != nil || claims == nil || claims.Issuer != "foodshop-refresh" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Invalid or expired refresh token",
			})
			return
		}
		// Optional: Prüfe, ob User noch existiert/aktiv ist
		user, err := db.GetUserByID(claims.UserID)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "User not found",
			})
			return
		}
		if !user.IsActive {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "User is not active",
			})
			return
		}
		// Neue Tokens generieren
		token, err := auth.GenerateToken(user.ID, user.Username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Failed to generate token",
			})
			return
		}
		refreshToken, err := auth.GenerateRefreshToken(user.ID, user.Username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Failed to generate refresh token",
			})
			return
		}
		resp := models.LoginResponse{
			Message:      "Token refreshed successfully",
			Token:        token,
			RefreshToken: refreshToken,
		}
		resp.User.ID = user.ID
		resp.User.Username = user.Username
		resp.User.Email = user.Email
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		// Removed the line 'resp' as it has no effect
	}
}
