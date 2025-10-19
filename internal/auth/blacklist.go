package auth

import (
	"sync"
	"time"
)

// TokenBlacklist manages invalidated tokens (for logout)
type TokenBlacklist struct {
	tokens map[string]time.Time
	mu     sync.RWMutex
}

// NewTokenBlacklist creates a new token blacklist
func NewTokenBlacklist() *TokenBlacklist {
	bl := &TokenBlacklist{
		tokens: make(map[string]time.Time),
	}

	// Start cleanup goroutine
	go bl.cleanup()

	return bl
}

// Add adds a token to the blacklist with expiration time
func (bl *TokenBlacklist) Add(token string, expiresAt time.Time) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.tokens[token] = expiresAt
}

// IsBlacklisted checks if a token is blacklisted
func (bl *TokenBlacklist) IsBlacklisted(token string) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	_, exists := bl.tokens[token]
	return exists
}

// cleanup removes expired tokens from the blacklist
func (bl *TokenBlacklist) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		bl.mu.Lock()
		now := time.Now()
		for token, expiresAt := range bl.tokens {
			if now.After(expiresAt) {
				delete(bl.tokens, token)
			}
		}
		bl.mu.Unlock()
	}
}
