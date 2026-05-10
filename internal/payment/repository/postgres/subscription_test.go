package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestSubscriptionRepo_UpsertActive(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewSubscriptionRepository(db)
	exp := time.Now().Add(30 * 24 * time.Hour)
	mock.ExpectExec(`INSERT INTO user_subscriptions`).
		WithArgs(int64(42), int64(1), int32(100000), exp).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.UpsertActive(context.Background(), 42, 1, 100000, exp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubscriptionRepo_GetActiveByUser_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewSubscriptionRepository(db)
	now := time.Now()
	exp := now.Add(30 * 24 * time.Hour)
	mock.ExpectQuery(`SELECT .+ FROM user_subscriptions us JOIN subscription_plans`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "plan_id", "name", "execution_remaining", "started_at", "expires_at", "created_at"}).
			AddRow(int64(1), int64(42), int64(1), "pro", int32(100000), now, exp, now))

	sub, err := repo.GetActiveByUser(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.PlanName != "pro" || sub.UserID != 42 {
		t.Fatalf("unexpected subscription: %+v", sub)
	}
}

func TestSubscriptionRepo_GetActiveByUser_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewSubscriptionRepository(db)
	mock.ExpectQuery(`SELECT .+ FROM user_subscriptions`).
		WithArgs(int64(99)).
		WillReturnError(sqlNoRows())

	_, err = repo.GetActiveByUser(context.Background(), 99)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
