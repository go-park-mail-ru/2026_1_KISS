package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestSessionRepo_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewSessionRepository(db)

	expiresAt := time.Now().Add(24 * time.Hour)

	mock.ExpectExec("INSERT INTO sessions").
		WithArgs("session-id-123", int64(1), expiresAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	session := &domain.Session{
		ID:        "session-id-123",
		UserID:    1,
		ExpiresAt: expiresAt,
	}

	err = repo.Create(context.Background(), session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSessionRepo_Create_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewSessionRepository(db)

	expiresAt := time.Now().Add(24 * time.Hour)
	dbErr := fmt.Errorf("connection refused")

	mock.ExpectExec("INSERT INTO sessions").
		WithArgs("session-id-123", int64(1), expiresAt).
		WillReturnError(dbErr)

	session := &domain.Session{
		ID:        "session-id-123",
		UserID:    1,
		ExpiresAt: expiresAt,
	}

	err = repo.Create(context.Background(), session)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSessionRepo_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewSessionRepository(db)

	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	rows := sqlmock.NewRows([]string{"id", "user_id", "expires_at", "created_at"}).
		AddRow("session-id-123", int64(1), expiresAt, now)

	mock.ExpectQuery("SELECT .+ FROM sessions WHERE id").
		WithArgs("session-id-123").
		WillReturnRows(rows)

	session, err := repo.GetByID(context.Background(), "session-id-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != "session-id-123" {
		t.Fatalf("expected id session-id-123, got %s", session.ID)
	}
	if session.UserID != 1 {
		t.Fatalf("expected user_id 1, got %d", session.UserID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSessionRepo_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewSessionRepository(db)

	mock.ExpectQuery("SELECT .+ FROM sessions WHERE id").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSessionRepo_DeleteByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewSessionRepository(db)

	mock.ExpectExec("DELETE FROM sessions WHERE id").
		WithArgs("session-id-123").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.DeleteByID(context.Background(), "session-id-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
