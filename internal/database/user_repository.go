package database

import (
	"database/sql"
	"errors"
	"fmt"
	"foodshop/internal/models"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrUserNotFound is returned when a user doesn't exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserExists is returned when trying to create a user that already exists.
	ErrUserExists = errors.New("user already exists")
	// ErrInvalidCredentials is returned when username or password is invalid.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrAccountLocked is returned when account is locked due to failed attempts.
	ErrAccountLocked = errors.New("account locked")
)

const (
	// MaxLoginAttempts is the maximum number of failed login attempts before lockout.
	MaxLoginAttempts = 5
	// LockoutDuration is how long an account stays locked.
	LockoutDuration = 15 * time.Minute
)

// UserRepository defines methods for user management.
type UserRepository interface {
	CreateUser(username, password, email string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	GetUserByID(id int64) (*models.User, error)
	DeleteUser(id int64) error
	DeactivateUser(id int64) error
	ActivateUser(id int64) error
	VerifyPassword(username, password string) (*models.User, error)
}

// CreateUser creates a new user with hashed password.
func (s *Sqlite) CreateUser(username, password, email string) (*models.User, error) {
	// Validate input
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	// Hash password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Insert user
	query := `
		INSERT INTO users (username, password, email, is_active)
		VALUES (?, ?, ?, 1)
	`
	result, err := s.db.Exec(query, username, string(hashedPassword), email)
	if err != nil {
		// Check for unique constraint violation (username already exists)
		if err.Error() == "UNIQUE constraint failed: users.username" {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	// Return the created user
	return s.GetUserByID(id)
}

// GetUserByUsername retrieves a user by username.
func (s *Sqlite) GetUserByUsername(username string) (*models.User, error) {
	query := `
		SELECT id, username, password, email, is_active, created_at, deactived_at,
		       failed_login_attempts, locked_until
		FROM users
		WHERE username = ?
	`

	user := &models.User{}
	var deactivedAt sql.NullTime
	var lockedUntil sql.NullTime

	err := s.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.Email,
		&user.IsActive,
		&user.CreatedAt,
		&deactivedAt,
		&user.FailedLoginAttempts,
		&lockedUntil,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	if deactivedAt.Valid {
		user.DeactivedAt = &deactivedAt.Time
	}
	if lockedUntil.Valid {
		user.LockedUntil = &lockedUntil.Time
	}

	return user, nil
}

// GetUserByID retrieves a user by ID.
func (s *Sqlite) GetUserByID(id int64) (*models.User, error) {
	query := `
		SELECT id, username, password, email, is_active, created_at, deactived_at,
		       failed_login_attempts, locked_until
		FROM users
		WHERE id = ?
	`

	user := &models.User{}
	var deactivedAt sql.NullTime
	var lockedUntil sql.NullTime

	err := s.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.Email,
		&user.IsActive,
		&user.CreatedAt,
		&deactivedAt,
		&user.FailedLoginAttempts,
		&lockedUntil,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	if deactivedAt.Valid {
		user.DeactivedAt = &deactivedAt.Time
	}
	if lockedUntil.Valid {
		user.LockedUntil = &lockedUntil.Time
	}

	return user, nil
}

// DeleteUser permanently deletes a user from the database.
// Note: This is a hard delete. Consider using DeactivateUser for soft deletes.
func (s *Sqlite) DeleteUser(id int64) error {
	query := `DELETE FROM users WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// DeactivateUser soft-deletes a user by setting is_active to false.
func (s *Sqlite) DeactivateUser(id int64) error {
	query := `
		UPDATE users
		SET is_active = 0, deactived_at = ?
		WHERE id = ? AND is_active = 1
	`

	result, err := s.db.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("deactivate user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// ActivateUser reactivates a deactivated user.
func (s *Sqlite) ActivateUser(id int64) error {
	query := `
		UPDATE users
		SET is_active = 1, deactived_at = NULL
		WHERE id = ? AND is_active = 0
	`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("activate user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// VerifyPassword checks if the provided password matches the stored hash.
// Returns the user if credentials are valid.
func (s *Sqlite) VerifyPassword(username, password string) (*models.User, error) {
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrInvalidCredentials
	}

	// Compare password with hash
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// IsAccountLocked checks if a user account is currently locked.
func (s *Sqlite) IsAccountLocked(username string) (bool, time.Time, error) {
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return false, time.Time{}, err
	}

	// If no lock time set, account is not locked
	if user.LockedUntil == nil {
		return false, time.Time{}, nil
	}

	// Check if lock has expired
	if time.Now().After(*user.LockedUntil) {
		// Lock expired, reset it
		s.UnlockAccount(username)
		return false, time.Time{}, nil
	}

	// Account is still locked
	return true, *user.LockedUntil, nil
}

// IncrementFailedAttempts increments the failed login counter for a user.
func (s *Sqlite) IncrementFailedAttempts(username string) error {
	query := `
		UPDATE users
		SET failed_login_attempts = failed_login_attempts + 1
		WHERE username = ?
	`

	_, err := s.db.Exec(query, username)
	if err != nil {
		return fmt.Errorf("increment failed attempts: %w", err)
	}

	return nil
}

// LockAccount locks a user account for the specified duration.
func (s *Sqlite) LockAccount(username string, duration time.Duration) error {
	lockUntil := time.Now().Add(duration)

	query := `
		UPDATE users
		SET locked_until = ?
		WHERE username = ?
	`

	result, err := s.db.Exec(query, lockUntil, username)
	if err != nil {
		return fmt.Errorf("lock account: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// ResetFailedAttempts resets the failed login counter to 0.
func (s *Sqlite) ResetFailedAttempts(username string) error {
	query := `
		UPDATE users
		SET failed_login_attempts = 0, locked_until = NULL
		WHERE username = ?
	`

	_, err := s.db.Exec(query, username)
	if err != nil {
		return fmt.Errorf("reset failed attempts: %w", err)
	}

	return nil
}

// UnlockAccount unlocks a user account and resets failed attempts.
func (s *Sqlite) UnlockAccount(username string) error {
	return s.ResetFailedAttempts(username)
}

// GetFailedAttempts returns the number of failed login attempts for a user.
func (s *Sqlite) GetFailedAttempts(username string) (int, error) {
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return 0, err
	}

	return user.FailedLoginAttempts, nil
}
