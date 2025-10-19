package handler

import (
	"encoding/json"
	"foodshop/internal/database"
	"foodshop/internal/middleware"
	"foodshop/internal/models"
	"log"
	"net/http"
)

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
