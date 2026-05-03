package httputil

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "user@example.com", false},
		{"valid with subdomain", "user@mail.example.com", false},
		{"no at sign", "userexample.com", true},
		{"no domain", "user@", true},
		{"no tld", "user@example", true},
		{"empty", "", true},

		{"valid with plus tag", "user+tag@example.com", false},
		{"valid with dots in local", "user.name@example.com", false},
		{"valid numeric local", "123@example.com", false},
		{"valid hyphen in domain", "user@my-site.com", false},
		{"valid local 64 chars", strings.Repeat("a", 64) + "@example.com", false},

		{"too long email over 255", "user@" + strings.Repeat("a", 250) + ".com", true},
		{"local part over 64 chars", strings.Repeat("a", 65) + "@example.com", true},
		{"double dot in local", "user..name@example.com", true},
		{"display name with angle brackets", "\"John\" <j@ex.com>", true},
		{"angle brackets only", "<j@ex.com>", true},
		{"IP literal in domain", "user@[127.0.0.1]", true},
		{"leading dot in domain", "user@.example.com", true},
		{"trailing dot in domain", "user@example.com.", true},
		{"leading hyphen in domain label", "user@-example.com", true},
		{"trailing hyphen in domain label", "user@example-.com", true},
		{"missing local part", "@example.com", true},
		{"double at sign", "user@@example.com", true},
		{"space in local", "user @example.com", true},
		{"single char TLD", "user@example.c", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid 8 chars", "password", false},
		{"valid long", "supersecretpassword123", false},
		{"too short 7", "pass123", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"valid", "user_123", false},
		{"valid min 3", "abc", false},
		{"valid max 50", "aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeee", false},
		{"too short 2", "ab", true},
		{"too long 51", "aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeeef", true},
		{"with dash", "user-name", true},
		{"with space", "user name", true},
		{"with dot", "user.name", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername(%q) error = %v, wantErr %v", tt.username, err, tt.wantErr)
			}
		})
	}
}
