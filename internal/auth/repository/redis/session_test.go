package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	redisv9 "github.com/redis/go-redis/v9"

	authredis "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository/redis"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestSessionRepo_CreateAndGetByID(t *testing.T) {
	ctx := context.Background()
	r, _, closeFn := newRepo(t)
	defer closeFn()

	session := &domain.Session{
		ID:        "sid-1",
		UserID:    42,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := r.Create(ctx, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := r.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.ID != session.ID {
		t.Fatalf("ID mismatch: got %q want %q", got.ID, session.ID)
	}
	if got.UserID != session.UserID {
		t.Fatalf("UserID mismatch: got %d want %d", got.UserID, session.UserID)
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set")
	}
}

func TestSessionRepo_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	r, _, closeFn := newRepo(t)
	defer closeFn()

	_, err := r.GetByID(ctx, "missing")
	if !errors.Is(err, domain.ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
}

func TestSessionRepo_GetByID_ExpiredByTTL(t *testing.T) {
	ctx := context.Background()
	r, mini, closeFn := newRepo(t)
	defer closeFn()

	session := &domain.Session{
		ID:        "sid-ttl",
		UserID:    101,
		ExpiresAt: time.Now().Add(2 * time.Second),
	}

	if err := r.Create(ctx, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	mini.FastForward(3 * time.Second)

	_, err := r.GetByID(ctx, session.ID)
	if !errors.Is(err, domain.ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired after ttl, got %v", err)
	}
}

func TestSessionRepo_DeleteByID(t *testing.T) {
	ctx := context.Background()
	r, _, closeFn := newRepo(t)
	defer closeFn()

	session := &domain.Session{
		ID:        "sid-2",
		UserID:    7,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := r.Create(ctx, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := r.DeleteByID(ctx, session.ID); err != nil {
		t.Fatalf("DeleteByID() error = %v", err)
	}
	_, err := r.GetByID(ctx, session.ID)
	if !errors.Is(err, domain.ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired after delete, got %v", err)
	}
}

func newRepo(t *testing.T) (*authredis.SessionRepo, *miniredis.Miniredis, func()) {
	t.Helper()

	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}

	client := redisv9.NewClient(&redisv9.Options{Addr: mini.Addr()})
	repo := authredis.NewSessionRepository(client)

	cleanup := func() {
		_ = client.Close()
		mini.Close()
	}

	return repo, mini, cleanup
}
