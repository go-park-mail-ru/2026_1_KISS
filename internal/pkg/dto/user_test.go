package dto

import (
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestNewUserResponse(t *testing.T) {
	now := time.Now()
	user := &domain.User{
		ID:          123,
		Username:    "testuser",
		Email:       "test@example.com",
		AvatarURL:   "https://example.com/avatar.jpg",
		Status:      "online",
		Description: "test description",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resp := NewUserResponse(user)

	if resp.ID != user.ID {
		t.Errorf("expected ID %d, got %d", user.ID, resp.ID)
	}
	if resp.Username != user.Username {
		t.Errorf("expected username %s, got %s", user.Username, resp.Username)
	}
	if resp.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, resp.Email)
	}
	if resp.AvatarURL != user.AvatarURL {
		t.Errorf("expected avatar URL %s, got %s", user.AvatarURL, resp.AvatarURL)
	}
	if resp.Status != user.Status {
		t.Errorf("expected status %s, got %s", user.Status, resp.Status)
	}
	if resp.Description != user.Description {
		t.Errorf("expected description %s, got %s", user.Description, resp.Description)
	}
	if !resp.CreatedAt.Equal(user.CreatedAt) {
		t.Errorf("expected created_at %v, got %v", user.CreatedAt, resp.CreatedAt)
	}
	if !resp.UpdatedAt.Equal(user.UpdatedAt) {
		t.Errorf("expected updated_at %v, got %v", user.UpdatedAt, resp.UpdatedAt)
	}
}

func TestNewUserResponse_EmptyFields(t *testing.T) {
	user := &domain.User{
		ID:       456,
		Username: "empty",
		Email:    "empty@example.com",
	}

	resp := NewUserResponse(user)

	if resp.ID != 456 {
		t.Errorf("expected ID 456, got %d", resp.ID)
	}
	if resp.AvatarURL != "" {
		t.Errorf("expected empty avatar URL, got %s", resp.AvatarURL)
	}
	if resp.Status != "" {
		t.Errorf("expected empty status, got %s", resp.Status)
	}
	if resp.Description != "" {
		t.Errorf("expected empty description, got %s", resp.Description)
	}
}
