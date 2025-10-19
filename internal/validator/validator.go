package validator

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	// Username must be alphanumeric with underscores only
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	// Email validation (basic but effective)
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// ValidateUsername validates username according to security best practices.
func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)

	if username == "" {
		return ValidationError{Field: "username", Message: "Username is required"}
	}

	if len(username) < 3 {
		return ValidationError{Field: "username", Message: "Username must be at least 3 characters long"}
	}

	if len(username) > 50 {
		return ValidationError{Field: "username", Message: "Username must not exceed 50 characters"}
	}

	if !usernameRegex.MatchString(username) {
		return ValidationError{Field: "username", Message: "Username can only contain letters, numbers, and underscores"}
	}

	// Prevent usernames that could be confused with system names
	reserved := []string{"admin", "root", "system", "api", "user", "guest", "test"}
	lowerUsername := strings.ToLower(username)
	for _, r := range reserved {
		if lowerUsername == r {
			return ValidationError{Field: "username", Message: "Username is reserved"}
		}
	}

	return nil
}

// ValidatePassword validates password strength.
func ValidatePassword(password string) error {
	if password == "" {
		return ValidationError{Field: "password", Message: "Password is required"}
	}

	if len(password) < 8 {
		return ValidationError{Field: "password", Message: "Password must be at least 8 characters long"}
	}

	if len(password) > 128 {
		return ValidationError{Field: "password", Message: "Password must not exceed 128 characters"}
	}

	// Check for at least one uppercase letter
	hasUpper := false
	// Check for at least one lowercase letter
	hasLower := false
	// Check for at least one digit
	hasDigit := false
	// Check for at least one special character
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return ValidationError{Field: "password", Message: "Password must contain at least one uppercase letter"}
	}

	if !hasLower {
		return ValidationError{Field: "password", Message: "Password must contain at least one lowercase letter"}
	}

	if !hasDigit {
		return ValidationError{Field: "password", Message: "Password must contain at least one digit"}
	}

	if !hasSpecial {
		return ValidationError{Field: "password", Message: "Password must contain at least one special character"}
	}

	return nil
}

// ValidateEmail validates email format.
func ValidateEmail(email string) error {
	if email == "" {
		return nil // Email is optional
	}

	email = strings.TrimSpace(email)

	if len(email) > 254 {
		return ValidationError{Field: "email", Message: "Email address is too long"}
	}

	if !emailRegex.MatchString(email) {
		return ValidationError{Field: "email", Message: "Invalid email format"}
	}

	return nil
}

// SanitizeInput removes potentially dangerous characters from input.
func SanitizeInput(input string) string {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Remove null bytes (can cause issues in C-based systems)
	input = strings.ReplaceAll(input, "\x00", "")

	// Remove control characters except newline and tab
	var sanitized strings.Builder
	for _, r := range input {
		if r >= 32 || r == '\n' || r == '\t' {
			sanitized.WriteRune(r)
		}
	}

	return sanitized.String()
}
