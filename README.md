# Foodshop - User Management API

## Overview

This application provides a user management system with SQLite database backend, implementing the Repository Pattern for easy database switching.

## Features

- ✅ User Registration with password hashing (bcrypt)
- ✅ User Authentication (JWT, Account Lockout)
- ✅ Account Lockout after failed login attempts
- ✅ Logout with token blacklist
- ✅ Soft Delete (Deactivate Users)
- ✅ Hard Delete (Permanent User Removal)
- ✅ User Reactivation
- ✅ Repository Pattern for database abstraction
- ✅ All handlers modular in `internal/handler/`

## API Endpoints

### Registration

**Endpoint:** `POST /registration`

**Request Body:**
```json
{
  "username": "johndoe",
  "password": "securePassword123",
  "password_verification": "securePassword123",
  "email": "john@example.com"
}
```

**Validation Rules:**
- Username: Required, 3-50 characters
- Password: Required, minimum 8 characters
- Password Verification: Required, must match password exactly
- Email: Optional, but must be valid if provided

**Success Response (201 Created):**
```json
{
  "message": "User created successfully",
  "user": {
    "id": 1,
    "username": "johndoe",
    "email": "john@example.com"
  }
}
```

**Error Responses:**

*400 Bad Request - Invalid data:*
```json
{
  "message": "Username is required"
}
```

*409 Conflict - Username already exists:*
```json
{
  "message": "Username already exists"
}
```

### Login

**Endpoint:** `POST /login`

**Request Body:**
```json
{
  "username": "johndoe",
  "password": "securePassword123"
}
```

**Security:**
- Account lockout after 5 failed attempts (15 min)
- JWT authentication
- All OWASP Priority 1 features implemented

## Database Schema

### Users Table

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,           -- bcrypt hashed
    email TEXT,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deactived_at DATETIME
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
```

## Repository Methods

The `Sqlite` struct implements the following methods:

### User Management

- `CreateUser(username, password, email string) (*models.User, error)`
  - Creates a new user with hashed password
  - Returns `ErrUserExists` if username is taken

- `GetUserByUsername(username string) (*models.User, error)`
  - Retrieves user by username
  - Returns `ErrUserNotFound` if not found

- `GetUserByID(id int64) (*models.User, error)`
  - Retrieves user by ID
  - Returns `ErrUserNotFound` if not found

- `DeleteUser(id int64) error`
  - Permanently deletes a user (hard delete)
  - Returns `ErrUserNotFound` if not found

- `DeactivateUser(id int64) error`
  - Soft deletes a user by setting `is_active = false`
  - Sets `deactived_at` timestamp
  - Returns `ErrUserNotFound` if not found

- `ActivateUser(id int64) error`
  - Reactivates a deactivated user
  - Clears `deactived_at` timestamp
  - Returns `ErrUserNotFound` if not found

- `VerifyPassword(username, password string) (*models.User, error)`
  - Verifies user credentials
  - Returns `ErrInvalidCredentials` for wrong password or inactive user
  - Returns `ErrUserNotFound` if user doesn't exist

## Running the Application

### Start the Server

```bash
go run cmd/web/main.go
```

The server will start on `127.0.0.1:8080` and automatically:
- Create the database file at `./data/foodshop.db`
- Initialize the schema if needed

### Testing

Run all tests:
```bash
go test ./...
```

Run database tests only:
```bash
go test -v ./internal/database/
```

## Security Features

1. **Password Hashing:** All passwords are hashed using bcrypt (cost factor 10)
2. **Unique Usernames:** Database constraint prevents duplicate usernames
3. **Input Validation:** Server-side validation for all user inputs
4. **Soft Deletes:** Users can be deactivated instead of permanently deleted
5. **Active User Check:** Deactivated users cannot authenticate
6. **Rate Limiting:** 10 req/sec, burst 20
7. **Security Headers:** OWASP recommended headers
8. **CORS:** Configurable, secure defaults
9. **Request Size/Timeout:** 1MB, 30s
10. **Account Lockout:** 5 failed logins = 15 min lock
11. **JWT Auth:** Stateless, secure
12. **Token Blacklist:** Secure logout

## Testing the Registration Endpoint

### Using curl:

```bash
# Create a new user
curl -X POST http://127.0.0.1:8080/registration \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "securepass123",
    "password_verification": "securepass123",
    "email": "test@example.com"
  }'

# Try to create duplicate user (should fail)
curl -X POST http://127.0.0.1:8080/registration \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "differentpass",
    "email": "other@example.com"
  }'
```

### Using httpie

```bash
# Create a new user
http POST :8080/registration \
  username=testuser \
  password=securepass123 \
  password_verification=securepass123 \
  email=test@example.com
```

## Project Structure

```
foodshop/
├── cmd/
│   └── web/
│       └── main.go              # Server setup, only routing
├── internal/
│   ├── database/
│   │   ├── database.go          # Core repository interface and Sqlite implementation
│   │   ├── database_test.go     # Database connection tests
│   │   ├── user_repository.go   # User-specific repository methods
│   │   └── user_repository_test.go  # User repository tests
│   ├── handler/
│   │   ├── user.go              # Profile & Index handlers
│   │   └── auth.go              # Login, Registration, Logout handlers
│   ├── models/
│   │   └── user.go              # User data models
│   ├── middleware/              # Security, rate limit, logging, etc.
│   └── validator/               # Input validation
├── data/
│   └── foodshop.db              # SQLite database (created automatically)
├── go.mod
├── go.sum
└── README.md
```

## Status

All core features und OWASP Priority 1 sind vollständig implementiert und getestet.
Account Lockout, JWT, Token Blacklist und alle Security-Middleware sind produktionsreif.

## Example Code

Siehe die Handler in `internal/handler/` für alle Endpunkte und die Middleware in `internal/middleware/` für Security-Features.

## License

MIT
