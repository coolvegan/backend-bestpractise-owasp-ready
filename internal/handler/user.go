package handler

import (
	"encoding/json"
	"foodshop/internal/database"
	"foodshop/internal/middleware"
	"foodshop/internal/models"
	"log"
	"net/http"
)

// UpdateUserRequest repräsentiert das Update-Request-Objekt
type UpdateUserRequest struct {
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
}

// UpdateUserHandler aktualisiert Passwort (optional) und E-Mail für den eingeloggten User
func UpdateUserHandler(db *database.Sqlite) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" && r.Method != "PATCH" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		userID, ok := middleware.GetUserID(r)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Unauthorized",
			})
			return
		}

		user, err := db.GetUserByID(userID)
		if err != nil {
			log.Printf("UpdateUserHandler: user not found: %v", err)
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "User not found",
			})
			return
		}

		var req UpdateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Invalid request body",
			})
			return
		}

		updated, err := db.UpdateUser(user.Username, req.Password, req.Email)
		if err != nil {
			log.Printf("UpdateUserHandler: update failed: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Update failed",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "User updated successfully",
			"user": map[string]interface{}{
				"id":         updated.ID,
				"username":   updated.Username,
				"email":      updated.Email,
				"is_active":  updated.IsActive,
				"created_at": updated.CreatedAt,
			},
		})
	}
}

// IndexHandler returns a welcome message
func IndexHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Welcome to Foodshop API",
			"version": "1.0.0",
		})
	}
}

// ProfileHandler returns the authenticated user's profile (protected endpoint)
func ProfileHandler(db *database.Sqlite) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		userID, ok := middleware.GetUserID(r)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Unauthorized",
			})
			return
		}
		user, err := db.GetUserByID(userID)
		if err != nil {
			log.Printf("Error fetching user profile: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrUserLogin{
				Message: "Failed to fetch profile",
			})
			return
		}
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
}

// ... weitere Handler folgen ...
