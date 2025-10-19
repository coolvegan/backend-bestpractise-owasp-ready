package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"foodshop/internal/database"
	"foodshop/internal/handler"
)

// setupTestDB creates a temporary database for testing.
func setupTestDB(t *testing.T) *database.Sqlite {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	testDB, ok := repo.(*database.Sqlite)
	if !ok {
		t.Fatal("Failed to cast repository to Sqlite")
	}

	if err := testDB.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	return testDB
}

func TestRegistrationHandler_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	payload := map[string]string{
		"username":              "testuser",
		"password":              "MyP@ssw0rd123",
		"password_verification": "MyP@ssw0rd123",
		"email":                 "test@example.com",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/registration", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.RegistrationHandler(db)(w, req)

	// Assert
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["message"] != "User created successfully" {
		t.Errorf("Expected success message, got: %v", response["message"])
	}
}

func TestRegistrationHandler_PasswordMismatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	payload := map[string]string{
		"username":              "testuser",
		"password":              "MyP@ssw0rd123",
		"password_verification": "Different_P@ss123",
		"email":                 "test@example.com",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/registration", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.RegistrationHandler(db)(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["message"] != "Passwords do not match" {
		t.Errorf("Expected 'Passwords do not match', got: %v", response["message"])
	}
}

func TestRegistrationHandler_MissingPasswordVerification(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	payload := map[string]string{
		"username": "testuser",
		"password": "MyP@ssw0rd123",
		"email":    "test@example.com",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/registration", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.RegistrationHandler(db)(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["message"] != "Password verification is required" {
		t.Errorf("Expected 'Password verification is required', got: %v", response["message"])
	}
}

func TestRegistrationHandler_ShortPassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	payload := map[string]string{
		"username":              "testuser",
		"password":              "Short1!",
		"password_verification": "Short1!",
		"email":                 "test@example.com",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/registration", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.RegistrationHandler(db)(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["message"] != "Password must be at least 8 characters long" {
		t.Errorf("Expected password length error, got: %v", response["message"])
	}
}

func TestRegistrationHandler_DuplicateUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	payload := map[string]string{
		"username":              "testuser",
		"password":              "MyP@ssw0rd123",
		"password_verification": "MyP@ssw0rd123",
		"email":                 "test@example.com",
	}
	body, _ := json.Marshal(payload)
	req1 := httptest.NewRequest("POST", "/registration", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.RegistrationHandler(db)(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("First registration failed: %s", w1.Body.String())
	}

	// Try to create duplicate user
	body2, _ := json.Marshal(payload)
	req2 := httptest.NewRequest("POST", "/registration", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.RegistrationHandler(db)(w2, req2)

	// Assert
	if w2.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w2.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w2.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["message"] != "Username already exists" {
		t.Errorf("Expected 'Username already exists', got: %v", response["message"])
	}
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestLoginHandler_AccountLockout(t *testing.T) {
	db := setupTestDB(t)
	// Erstellt einen User
	_, err := db.CreateUser("lockuser", "LockP@ssw0rd!", "lock@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	loginHandler := handler.LoginHandler(db)

	// 1. Erfolgreicher Login
	login := map[string]string{"username": "lockuser", "password": "LockP@ssw0rd!"}
	body, _ := json.Marshal(login)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	loginHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}

	// 2. Mehrfache Fehlversuche
	for i := 1; i <= database.MaxLoginAttempts; i++ {
		badLogin := map[string]string{"username": "lockuser", "password": "Falsch123!"}
		body, _ := json.Marshal(badLogin)
		req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		loginHandler(w, req)
		if i < database.MaxLoginAttempts && w.Code != http.StatusUnauthorized {
			t.Errorf("Attempt %d: Expected 401 Unauthorized, got %d", i, w.Code)
		}
		if i == database.MaxLoginAttempts && w.Code != http.StatusLocked {
			t.Errorf("Attempt %d: Expected 423 Locked, got %d", i, w.Code)
		}
	}

	// 3. Gesperrtes Konto liefert 423
	badLogin := map[string]string{"username": "lockuser", "password": "LockP@ssw0rd!"}
	body, _ = json.Marshal(badLogin)
	req = httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	loginHandler(w, req)
	if w.Code != http.StatusLocked {
		t.Errorf("Locked account: Expected 423 Locked, got %d", w.Code)
	}

	// 4. Nach Ablauf der Sperrzeit ist Login wieder möglich
	db.UnlockAccount("lockuser") // Simuliere Ablauf der Sperrzeit
	body, _ = json.Marshal(login)
	req = httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	loginHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("After unlock: Expected 200 OK, got %d", w.Code)
	}
}
