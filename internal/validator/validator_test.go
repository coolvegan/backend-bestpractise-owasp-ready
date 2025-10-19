package validator

import (
	"testing"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		wantError bool
	}{
		{"Valid username", "john_doe123", false},
		{"Valid short", "abc", false},
		{"Too short", "ab", true},
		{"Too long", "this_is_a_very_long_username_that_exceeds_fifty_chars_limit", true},
		{"Empty", "", true},
		{"With spaces", "john doe", true},
		{"With special chars", "john@doe", true},
		{"With hyphen", "john-doe", true},
		{"Reserved admin", "admin", true},
		{"Reserved root", "root", true},
		{"Valid with underscore", "user_name_123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateUsername(%q) error = %v, wantError %v", tt.username, err, tt.wantError)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		wantError bool
	}{
		{"Valid strong password", "MyP@ssw0rd!", false},
		{"Too short", "Pass1!", true},
		{"No uppercase", "myp@ssw0rd!", true},
		{"No lowercase", "MYP@SSW0RD!", true},
		{"No digit", "MyPassword!", true},
		{"No special char", "MyPassword1", true},
		{"Empty", "", true},
		{"Valid minimum", "Abcd123!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePassword() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		wantError bool
	}{
		{"Valid email", "test@example.com", false},
		{"Valid with subdomain", "user@mail.example.com", false},
		{"Valid with plus", "user+tag@example.com", false},
		{"Empty (optional)", "", false},
		{"Invalid no @", "testexample.com", true},
		{"Invalid no domain", "test@", true},
		{"Invalid no TLD", "test@example", true},
		{"Invalid spaces", "test @example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateEmail(%q) error = %v, wantError %v", tt.email, err, tt.wantError)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Normal text", "Hello World", "Hello World"},
		{"With whitespace", "  Hello  ", "Hello"},
		{"With null bytes", "Hello\x00World", "HelloWorld"},
		{"With control chars", "Hello\x01\x02World", "HelloWorld"},
		{"With newline (allowed)", "Hello\nWorld", "Hello\nWorld"},
		{"With tab (allowed)", "Hello\tWorld", "Hello\tWorld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeInput(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeInput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
