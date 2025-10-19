package database

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNew verifies that New() creates a repository and connects to sqlite.
func TestNew(t *testing.T) {
	// Create a temporary directory for test databases
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	// Verify we can ping the database
	if err := repo.DB().Ping(); err != nil {
		t.Errorf("Ping() failed: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

// TestNewEmptyPath verifies that New() rejects empty paths.
func TestNewEmptyPath(t *testing.T) {
	_, err := New("")
	if err == nil {
		t.Error("New(\"\") should return an error")
	}
}

// TestInitSchema verifies that InitSchema creates the expected tables.
func TestInitSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_schema.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	// Cast to *Sqlite to access InitSchema
	sqliteRepo, ok := repo.(*Sqlite)
	if !ok {
		t.Fatal("Repository is not *Sqlite")
	}

	// Initialize the schema
	if err := sqliteRepo.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Verify the users table exists by querying sqlite_master
	var tableName string
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name='users'"
	err = repo.DB().QueryRow(query).Scan(&tableName)
	if err != nil {
		t.Fatalf("Failed to query sqlite_master: %v", err)
	}

	if tableName != "users" {
		t.Errorf("Expected table 'users', got '%s'", tableName)
	}

	// Verify the index exists
	var indexName string
	indexQuery := "SELECT name FROM sqlite_master WHERE type='index' AND name='idx_users_username'"
	err = repo.DB().QueryRow(indexQuery).Scan(&indexName)
	if err != nil {
		t.Fatalf("Failed to query for index: %v", err)
	}

	if indexName != "idx_users_username" {
		t.Errorf("Expected index 'idx_users_username', got '%s'", indexName)
	}
}

// TestInitSchemaIdempotent verifies that InitSchema can be called multiple times.
func TestInitSchemaIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_idempotent.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	sqliteRepo := repo.(*Sqlite)

	// Call InitSchema twice
	if err := sqliteRepo.InitSchema(); err != nil {
		t.Fatalf("First InitSchema() failed: %v", err)
	}

	if err := sqliteRepo.InitSchema(); err != nil {
		t.Fatalf("Second InitSchema() failed: %v", err)
	}
}

// TestInsertUser verifies we can insert a user after schema initialization.
func TestInsertUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_insert.db")

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer repo.Close()

	sqliteRepo := repo.(*Sqlite)
	if err := sqliteRepo.InitSchema(); err != nil {
		t.Fatalf("InitSchema() failed: %v", err)
	}

	// Insert a test user
	insertSQL := "INSERT INTO users (username, password) VALUES (?, ?)"
	result, err := repo.DB().Exec(insertSQL, "testuser", "hashedpassword123")
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	// Verify the insert
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get LastInsertId: %v", err)
	}

	if id != 1 {
		t.Errorf("Expected first insert ID to be 1, got %d", id)
	}

	// Verify we can read it back
	var username, password string
	querySQL := "SELECT username, password FROM users WHERE id = ?"
	err = repo.DB().QueryRow(querySQL, id).Scan(&username, &password)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}

	if username != "testuser" || password != "hashedpassword123" {
		t.Errorf("Expected (testuser, hashedpassword123), got (%s, %s)", username, password)
	}
}
