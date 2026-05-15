package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestOAuthAccountRepo_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewOAuthAccountRepository(db)
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO oauth_accounts`).
		WithArgs(int64(1), "google", "g-42").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(7), now))

	acc := &domain.OAuthAccount{UserID: 1, Provider: "google", ProviderID: "g-42"}
	id, err := repo.Create(context.Background(), acc)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id != 7 || acc.ID != 7 {
		t.Fatalf("want id=7, got id=%d acc.ID=%d", id, acc.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthAccountRepo_Create_Conflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewOAuthAccountRepository(db)

	mock.ExpectQuery(`INSERT INTO oauth_accounts`).
		WithArgs(int64(1), "google", "g-42").
		WillReturnError(&pq.Error{Code: "23505"})

	_, err = repo.Create(context.Background(), &domain.OAuthAccount{UserID: 1, Provider: "google", ProviderID: "g-42"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestOAuthAccountRepo_GetByProviderID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewOAuthAccountRepository(db)
	now := time.Now()

	mock.ExpectQuery(`SELECT .+ FROM oauth_accounts WHERE provider`).
		WithArgs("google", "g-42").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "provider", "provider_id", "created_at"}).
			AddRow(int64(7), int64(1), "google", "g-42", now))

	acc, err := repo.GetByProviderID(context.Background(), "google", "g-42")
	if err != nil {
		t.Fatalf("GetByProviderID: %v", err)
	}
	if acc.UserID != 1 || acc.Provider != "google" {
		t.Fatalf("unexpected: %+v", acc)
	}
}

func TestOAuthAccountRepo_GetByProviderID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewOAuthAccountRepository(db)

	mock.ExpectQuery(`SELECT .+ FROM oauth_accounts WHERE provider`).
		WithArgs("yandex", "missing").
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetByProviderID(context.Background(), "yandex", "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestOAuthAccountRepo_ListByUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewOAuthAccountRepository(db)
	now := time.Now()

	mock.ExpectQuery(`SELECT .+ FROM oauth_accounts WHERE user_id`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "provider", "provider_id", "created_at"}).
			AddRow(int64(7), int64(1), "google", "g-1", now).
			AddRow(int64(8), int64(1), "yandex", "y-1", now))

	list, err := repo.ListByUserID(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListByUserID: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 accounts, got %d", len(list))
	}
}

func TestOAuthAccountRepo_DeleteByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewOAuthAccountRepository(db)

	mock.ExpectExec(`DELETE FROM oauth_accounts`).
		WithArgs(int64(99), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.DeleteByID(context.Background(), 99, 1)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
