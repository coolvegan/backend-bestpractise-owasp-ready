package database

import (
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_update_user.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	db := repo.(*Sqlite)
	if err := db.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Create a user
	user, err := db.CreateUser("testuser", "password123", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser() failed: %v", err)
	}

	// Update only email
	updated, err := db.UpdateUser("testuser", "", "new@example.com")
	if err != nil {
		t.Fatalf("UpdateUser() failed: %v", err)
	}
	if updated.Email != "new@example.com" {
		t.Errorf("Expected email 'new@example.com', got '%s'", updated.Email)
	}
	if updated.Password != user.Password {
		t.Error("Password should not change if empty string is given")
	}

	// Update password and email
	updated2, err := db.UpdateUser("testuser", "newpass456", "another@example.com")
	if err != nil {
		t.Fatalf("UpdateUser() failed: %v", err)
	}
	if updated2.Email != "another@example.com" {
		t.Errorf("Expected email 'another@example.com', got '%s'", updated2.Email)
	}
	if updated2.Password == user.Password {
		t.Error("Password hash should change when password is updated")
	}

	// Password should be hashed
	if updated2.Password == "newpass456" {
		t.Error("Password should be hashed, not stored in plain text")
	}

	// Update non-existent user
	_, err = db.UpdateUser("doesnotexist", "irrelevant", "irrelevant@example.com")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound for non-existent user, got %v", err)
	}
}

// TestCreateUser verifies user creation with password hashing.
func TestCreateUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_users.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	db := repo.(*Sqlite)
	if err := db.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Create a user
	user, err := db.CreateUser("testuser", "password123", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser() failed: %v", err)
	}

	// Verify user fields
	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", user.Email)
	}
	if !user.IsActive {
		t.Error("Expected user to be active")
	}
	if user.Password == "password123" {
		t.Error("Password should be hashed, not stored in plain text")
	}
}

// TestCreateUserDuplicate verifies that duplicate usernames are rejected.
func TestCreateUserDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_duplicate.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	db := repo.(*Sqlite)
	if err := db.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Create first user
	_, err = db.CreateUser("testuser", "password123", "test@example.com")
	if err != nil {
		t.Fatalf("First CreateUser() failed: %v", err)
	}

	// Try to create duplicate user
	_, err = db.CreateUser("testuser", "different_password", "different@example.com")
	if err != ErrUserExists {
		t.Errorf("Expected ErrUserExists, got %v", err)
	}
}

// TestGetUserByUsername verifies user retrieval by username.
func TestGetUserByUsername(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_get_user.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	db := repo.(*Sqlite)
	if err := db.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Create a user
	created, err := db.CreateUser("testuser", "password123", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser() failed: %v", err)
	}

	// Retrieve user
	user, err := db.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("GetUserByUsername() failed: %v", err)
	}

	if user.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, user.ID)
	}

	// Try to get non-existent user
	_, err = db.GetUserByUsername("nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

// TestDeleteUser verifies user deletion.
func TestDeleteUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_delete.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	db := repo.(*Sqlite)
	if err := db.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Create a user
	user, err := db.CreateUser("testuser", "password123", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser() failed: %v", err)
	}

	// Delete the user
	if err := db.DeleteUser(user.ID); err != nil {
		t.Fatalf("DeleteUser() failed: %v", err)
	}

	// Verify user is gone
	_, err = db.GetUserByID(user.ID)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound after deletion, got %v", err)
	}

	// Try to delete non-existent user
	err = db.DeleteUser(999)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound for non-existent user, got %v", err)
	}
}

// TestDeactivateUser verifies user deactivation (soft delete).
func TestDeactivateUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_deactivate.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	db := repo.(*Sqlite)
	if err := db.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Create a user
	user, err := db.CreateUser("testuser", "password123", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser() failed: %v", err)
	}

	// Deactivate the user
	if err := db.DeactivateUser(user.ID); err != nil {
		t.Fatalf("DeactivateUser() failed: %v", err)
	}

	// Verify user is deactivated
	deactivated, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID() after deactivation failed: %v", err)
	}

	if deactivated.IsActive {
		t.Error("Expected user to be inactive")
	}
	if deactivated.DeactivedAt == nil {
		t.Error("Expected deactived_at to be set")
	}
}

// TestActivateUser verifies user reactivation.
func TestActivateUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_activate.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	db := repo.(*Sqlite)
	if err := db.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Create and deactivate a user
	user, err := db.CreateUser("testuser", "password123", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser() failed: %v", err)
	}

	if err := db.DeactivateUser(user.ID); err != nil {
		t.Fatalf("DeactivateUser() failed: %v", err)
	}

	// Reactivate the user
	if err := db.ActivateUser(user.ID); err != nil {
		t.Fatalf("ActivateUser() failed: %v", err)
	}

	// Verify user is active
	activated, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID() after activation failed: %v", err)
	}

	if !activated.IsActive {
		t.Error("Expected user to be active")
	}
	if activated.DeactivedAt != nil {
		t.Error("Expected deactived_at to be NULL")
	}
}

// TestVerifyPassword verifies password verification.
func TestVerifyPassword(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_verify.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	db := repo.(*Sqlite)
	if err := db.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Create a user
	_, err = db.CreateUser("testuser", "password123", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser() failed: %v", err)
	}

	// Verify correct password
	user, err := db.VerifyPassword("testuser", "password123")
	if err != nil {
		t.Fatalf("VerifyPassword() with correct password failed: %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}

	// Verify wrong password
	_, err = db.VerifyPassword("testuser", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials for wrong password, got %v", err)
	}

	// Verify non-existent user
	_, err = db.VerifyPassword("nonexistent", "password123")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound for non-existent user, got %v", err)
	}
}

// TestVerifyPasswordDeactivatedUser verifies that deactivated users can't login.
func TestVerifyPasswordDeactivatedUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_verify_deactivated.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	db := repo.(*Sqlite)
	if err := db.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Create and deactivate a user
	user, err := db.CreateUser("testuser", "password123", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser() failed: %v", err)
	}

	if err := db.DeactivateUser(user.ID); err != nil {
		t.Fatalf("DeactivateUser() failed: %v", err)
	}

	// Try to verify password for deactivated user
	_, err = db.VerifyPassword("testuser", "password123")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials for deactivated user, got %v", err)
	}
}

// TestAccountLockout verifies account lockout after failed attempts.
func TestAccountLockout(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_lockout.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	db := repo.(*Sqlite)
	if err := db.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Create a user
	_, err = db.CreateUser("testuser", "password123", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser() failed: %v", err)
	}

	// Initially not locked
	isLocked, _, err := db.IsAccountLocked("testuser")
	if err != nil {
		t.Fatalf("IsAccountLocked() failed: %v", err)
	}
	if isLocked {
		t.Error("Account should not be locked initially")
	}

	// Increment failed attempts
	for i := 1; i < MaxLoginAttempts; i++ {
		if err := db.IncrementFailedAttempts("testuser"); err != nil {
			t.Fatalf("IncrementFailedAttempts() failed: %v", err)
		}

		attempts, err := db.GetFailedAttempts("testuser")
		if err != nil {
			t.Fatalf("GetFailedAttempts() failed: %v", err)
		}
		if attempts != i {
			t.Errorf("Expected %d failed attempts, got %d", i, attempts)
		}
	}

	// Lock account manually
	if err := db.LockAccount("testuser", LockoutDuration); err != nil {
		t.Fatalf("LockAccount() failed: %v", err)
	}

	// Verify account is locked
	isLocked, lockedUntil, err := db.IsAccountLocked("testuser")
	if err != nil {
		t.Fatalf("IsAccountLocked() after lock failed: %v", err)
	}
	if !isLocked {
		t.Error("Account should be locked")
	}
	if time.Until(lockedUntil) > LockoutDuration {
		t.Errorf("Lock duration too long: %v", lockedUntil)
	}

	// Reset failed attempts
	if err := db.ResetFailedAttempts("testuser"); err != nil {
		t.Fatalf("ResetFailedAttempts() failed: %v", err)
	}

	attempts, err := db.GetFailedAttempts("testuser")
	if err != nil {
		t.Fatalf("GetFailedAttempts() after reset failed: %v", err)
	}
	if attempts != 0 {
		t.Errorf("Expected 0 failed attempts after reset, got %d", attempts)
	}

	// Unlock account
	if err := db.UnlockAccount("testuser"); err != nil {
		t.Fatalf("UnlockAccount() failed: %v", err)
	}

	// Verify account is unlocked
	isLocked, _, err = db.IsAccountLocked("testuser")
	if err != nil {
		t.Fatalf("IsAccountLocked() after unlock failed: %v", err)
	}
	if isLocked {
		t.Error("Account should be unlocked")
	}
}

// TestAccountLockoutExpiration verifies that locks expire automatically.
func TestAccountLockoutExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_lockout_expire.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	db := repo.(*Sqlite)
	if err := db.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Create a user
	_, err = db.CreateUser("testuser", "password123", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser() failed: %v", err)
	}

	// Lock account for 1 second
	if err := db.LockAccount("testuser", 1*time.Second); err != nil {
		t.Fatalf("LockAccount() failed: %v", err)
	}

	// Verify account is locked
	isLocked, _, err := db.IsAccountLocked("testuser")
	if err != nil {
		t.Fatalf("IsAccountLocked() failed: %v", err)
	}
	if !isLocked {
		t.Error("Account should be locked")
	}

	// Wait for lock to expire
	time.Sleep(2 * time.Second)

	// Verify account is automatically unlocked
	isLocked, _, err = db.IsAccountLocked("testuser")
	if err != nil {
		t.Fatalf("IsAccountLocked() after expiration failed: %v", err)
	}
	if isLocked {
		t.Error("Account should be unlocked after expiration")
	}
}

// TestAccountLockoutNonExistentUser verifies error handling for non-existent users.
func TestAccountLockoutNonExistentUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_lockout_nonexistent.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	db := repo.(*Sqlite)
	if err := db.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Try lockout operations on non-existent user
	// IsAccountLocked should return ErrUserNotFound
	_, _, err = db.IsAccountLocked("nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound for IsAccountLocked, got %v", err)
	}

	// LockAccount should return ErrUserNotFound
	err = db.LockAccount("nonexistent", LockoutDuration)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound for LockAccount, got %v", err)
	}

	// GetFailedAttempts should return ErrUserNotFound
	_, err = db.GetFailedAttempts("nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound for GetFailedAttempts, got %v", err)
	}

	// These operations are idempotent and succeed silently
	// IncrementFailedAttempts - no-op if user doesn't exist
	err = db.IncrementFailedAttempts("nonexistent")
	if err != nil {
		t.Errorf("IncrementFailedAttempts should succeed silently for non-existent user, got %v", err)
	}

	// ResetFailedAttempts - no-op if user doesn't exist
	err = db.ResetFailedAttempts("nonexistent")
	if err != nil {
		t.Errorf("ResetFailedAttempts should succeed silently for non-existent user, got %v", err)
	}

	// UnlockAccount - no-op if user doesn't exist
	err = db.UnlockAccount("nonexistent")
	if err != nil {
		t.Errorf("UnlockAccount should succeed silently for non-existent user, got %v", err)
	}
}
