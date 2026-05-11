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

func newStateRepo(t *testing.T) (*authredis.OAuthStateRepo, *miniredis.Miniredis, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	client := redisv9.NewClient(&redisv9.Options{Addr: mr.Addr()})
	return authredis.NewOAuthStateRepository(client), mr, func() {
		_ = client.Close()
		mr.Close()
	}
}

func TestOAuthStateRepo_SaveAndConsume(t *testing.T) {
	r, mr, closeFn := newStateRepo(t)
	defer closeFn()
	ctx := context.Background()

	st := &domain.OAuthState{
		State:        "state-1",
		Provider:     "google",
		CodeVerifier: "verifier-1",
	}
	if err := r.Save(ctx, st, 5*time.Minute); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := r.Consume(ctx, "state-1")
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if got.Provider != "google" || got.CodeVerifier != "verifier-1" {
		t.Fatalf("unexpected: %+v", got)
	}

	if _, err := r.Consume(ctx, "state-1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("second consume must be ErrNotFound, got %v", err)
	}

	_ = mr
}

func TestOAuthStateRepo_Save_ConflictOnDuplicate(t *testing.T) {
	r, _, closeFn := newStateRepo(t)
	defer closeFn()
	ctx := context.Background()

	st := &domain.OAuthState{State: "x", Provider: "google", CodeVerifier: "v"}
	if err := r.Save(ctx, st, time.Minute); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	if err := r.Save(ctx, st, time.Minute); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second Save must be ErrConflict, got %v", err)
	}
}

func TestOAuthStateRepo_Consume_EmptyState(t *testing.T) {
	r, _, closeFn := newStateRepo(t)
	defer closeFn()

	if _, err := r.Consume(context.Background(), ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestOAuthStateRepo_Save_ExpiresViaTTL(t *testing.T) {
	r, mr, closeFn := newStateRepo(t)
	defer closeFn()

	st := &domain.OAuthState{State: "ttl-state", Provider: "vkid", CodeVerifier: "v"}
	if err := r.Save(context.Background(), st, time.Minute); err != nil {
		t.Fatalf("Save: %v", err)
	}

	mr.FastForward(2 * time.Minute)

	if _, err := r.Consume(context.Background(), "ttl-state"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("after TTL expiry expected ErrNotFound, got %v", err)
	}
}
