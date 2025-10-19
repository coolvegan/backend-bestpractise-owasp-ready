package models

import "time"

// ErrUserLogin represents an error response for user login/registration.
type ErrUserLogin struct {
	Message string `json:"message"`
}

// UserLogin represents login credentials.
type UserLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserRegistration represents registration data.
type UserRegistration struct {
	Username             string `json:"username"`
	Password             string `json:"password"`
	PasswordVerification string `json:"password_verification"`
	Email                string `json:"email,omitempty"`
}

// User represents a user in the database.
type User struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Password    string     `json:"-"` // Never expose password in JSON
	Email       string     `json:"email,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	DeactivedAt *time.Time `json:"deactived_at,omitempty"`
}
