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

func TestEventRepo_CountActiveUsersByDay_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewEventRepository(db)

	since := time.Now().Add(-7 * 24 * time.Hour)
	day1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"day", "count"}).
		AddRow(day1, int64(10)).
		AddRow(day2, int64(20))

	mock.ExpectQuery("SELECT DATE").
		WithArgs(since).
		WillReturnRows(rows)

	result, err := repo.CountActiveUsersByDay(context.Background(), since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result[0].Count != 10 || result[1].Count != 20 {
		t.Fatalf("unexpected counts: %v, %v", result[0].Count, result[1].Count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEventRepo_CountActiveUsersByDay_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewEventRepository(db)

	since := time.Now().Add(-7 * 24 * time.Hour)

	mock.ExpectQuery("SELECT DATE").
		WithArgs(since).
		WillReturnError(errors.New("db error"))

	_, err = repo.CountActiveUsersByDay(context.Background(), since)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEventRepo_CountActiveUsersByDay_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewEventRepository(db)

	since := time.Now().Add(-7 * 24 * time.Hour)
	rows := sqlmock.NewRows([]string{"day", "count"})

	mock.ExpectQuery("SELECT DATE").
		WithArgs(since).
		WillReturnRows(rows)

	result, err := repo.CountActiveUsersByDay(context.Background(), since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(result))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEventRepo_CountActiveUsersByMonth_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewEventRepository(db)

	since := time.Now().Add(-30 * 24 * time.Hour)
	month1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	month2 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"month", "count"}).
		AddRow(month1, int64(100)).
		AddRow(month2, int64(200))

	mock.ExpectQuery("SELECT DATE_TRUNC").
		WithArgs(since).
		WillReturnRows(rows)

	result, err := repo.CountActiveUsersByMonth(context.Background(), since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result[0].Count != 100 || result[1].Count != 200 {
		t.Fatalf("unexpected counts: %v, %v", result[0].Count, result[1].Count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEventRepo_CountActiveUsersByMonth_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewEventRepository(db)

	since := time.Now().Add(-30 * 24 * time.Hour)

	mock.ExpectQuery("SELECT DATE_TRUNC").
		WithArgs(since).
		WillReturnError(errors.New("db error"))

	_, err = repo.CountActiveUsersByMonth(context.Background(), since)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEventRepo_CountActiveUsersByMonth_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewEventRepository(db)

	since := time.Now().Add(-30 * 24 * time.Hour)
	rows := sqlmock.NewRows([]string{"month", "count"})

	mock.ExpectQuery("SELECT DATE_TRUNC").
		WithArgs(since).
		WillReturnRows(rows)

	result, err := repo.CountActiveUsersByMonth(context.Background(), since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(result))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
