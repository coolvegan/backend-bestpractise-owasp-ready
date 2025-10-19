package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Repository defines the minimal interface the rest of the app depends on.
// This keeps the implementation swappable (sqlite, postgres, in-memory, ...).
type Repository interface {
	// DB returns the underlying *sql.DB for advanced usage.
	DB() *sql.DB
	// Close closes the underlying connection(s).
	Close() error
}

// Sqlite is a simple wrapper around *sql.DB for sqlite3.
type Sqlite struct {
	db *sql.DB
}

// New opens (or creates) a sqlite database at the provided path and returns
// a Repository. If the path is empty an error is returned.
// The sqlite3 driver will create the file if it does not exist.
func New(path string) (Repository, error) {
	if path == "" {
		return nil, fmt.Errorf("database path must not be empty")
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Verify the DB is reachable (will also create the file on disk for sqlite).
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	// Small sqlite tuning: single writer allowed.
	db.SetMaxOpenConns(1)

	return &Sqlite{db: db}, nil
}

// DB returns the underlying *sql.DB.
func (s *Sqlite) DB() *sql.DB { return s.db }

// Close closes the database connection.
func (s *Sqlite) Close() error { return s.db.Close() }

// InitSchema creates the initial database schema.
// Call this after New() to set up tables if they don't exist.
func (s *Sqlite) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		email TEXT,
		is_active BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		deactived_at DATETIME,
		failed_login_attempts INTEGER DEFAULT 0,
		locked_until DATETIME
	);
	
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("init schema: %w", err)
	}

	// Add columns to existing tables if they don't exist (migration)
	migrations := []string{
		`ALTER TABLE users ADD COLUMN failed_login_attempts INTEGER DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN locked_until DATETIME`,
	}

	for _, migration := range migrations {
		// Ignore errors if column already exists
		s.db.Exec(migration)
	}

	return nil
}
