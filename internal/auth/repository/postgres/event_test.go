package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestEventRepo_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewEventRepository(db)

	metadata := json.RawMessage("{}")
	mock.ExpectExec("INSERT INTO user_events").
		WithArgs(int64(1), "login", metadata).
		WillReturnResult(sqlmock.NewResult(1, 1))

	event := &domain.UserEvent{
		UserID:    1,
		EventType: "login",
		Metadata:  metadata,
	}

	err = repo.Create(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEventRepo_CountActiveUsers_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewEventRepository(db)

	since := time.Now().Add(-24 * time.Hour)
	rows := sqlmock.NewRows([]string{"count"}).AddRow(int64(42))

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(since).
		WillReturnRows(rows)

	count, err := repo.CountActiveUsers(context.Background(), since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 42 {
		t.Fatalf("expected count 42, got %d", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEventRepo_Create_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewEventRepository(db)

	metadata := json.RawMessage("{}")
	mock.ExpectExec("INSERT INTO user_events").
		WithArgs(int64(1), "login", metadata).
		WillReturnError(errors.New("db error"))

	event := &domain.UserEvent{
		UserID:    1,
		EventType: "login",
		Metadata:  metadata,
	}

	err = repo.Create(context.Background(), event)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEventRepo_CountActiveUsers_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewEventRepository(db)

	since := time.Now().Add(-24 * time.Hour)

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(since).
		WillReturnError(errors.New("db error"))

	_, err = repo.CountActiveUsers(context.Background(), since)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEventRepo_CountActiveUsers_Zero(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewEventRepository(db)

	since := time.Now().Add(-24 * time.Hour)
	rows := sqlmock.NewRows([]string{"count"}).AddRow(int64(0))

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(since).
		WillReturnRows(rows)

	count, err := repo.CountActiveUsers(context.Background(), since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected count 0, got %d", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
