package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func sqlNoRowsAuth() error { return sql.ErrNoRows }

func TestSubscriptionViewRepo_GetActive_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewSubscriptionViewRepository(db)
	exp := time.Now().Add(30 * 24 * time.Hour)
	mock.ExpectQuery(`SELECT .+ FROM user_subscriptions us JOIN subscription_plans`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "expires_at"}).
			AddRow(int64(1), "pro", exp))

	sub, err := repo.GetActive(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub == nil || sub.PlanName != "pro" {
		t.Fatalf("unexpected: %+v", sub)
	}
}

func TestSubscriptionViewRepo_GetActive_NoneReturnsNilNil(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewSubscriptionViewRepository(db)
	mock.ExpectQuery(`SELECT .+ FROM user_subscriptions`).
		WithArgs(int64(99)).
		WillReturnError(sqlNoRowsAuth())

	sub, err := repo.GetActive(context.Background(), 99)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if sub != nil {
		t.Errorf("expected nil subscription, got %+v", sub)
	}
}
