package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/lib/pq"
)

func TestUserRepo_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow(int64(1), now, now)

	mock.ExpectQuery("INSERT INTO users").
		WithArgs("testuser", "test@example.com", "hashedpwd").
		WillReturnRows(rows)

	user := &domain.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpwd",
	}

	id, err := repo.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 1 {
		t.Fatalf("expected id 1, got %d", id)
	}
	if user.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
	if user.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_Create_UniqueViolation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectQuery("INSERT INTO users").
		WithArgs("testuser", "test@example.com", "hashedpwd").
		WillReturnError(&pq.Error{Code: "23505"})

	user := &domain.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpwd",
	}

	_, err = repo.Create(context.Background(), user)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "username", "email", "password_hash",
		"avatar_url", "status", "description",
		"created_at", "updated_at",
	}).AddRow(int64(1), "testuser", "test@example.com", "hashedpwd", "avatar.png", "active", "desc", now, now)

	mock.ExpectQuery("SELECT .+ FROM users WHERE id").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	user, err := repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 1 {
		t.Fatalf("expected id 1, got %d", user.ID)
	}
	if user.Username != "testuser" {
		t.Fatalf("expected username testuser, got %s", user.Username)
	}
	if user.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", user.Email)
	}
	if user.AvatarURL != "avatar.png" {
		t.Fatalf("expected avatar_url avatar.png, got %s", user.AvatarURL)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectQuery("SELECT .+ FROM users WHERE id").
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetByID(context.Background(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_GetByEmail_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "username", "email", "password_hash",
		"avatar_url", "status", "description",
		"created_at", "updated_at",
	}).AddRow(int64(1), "testuser", "test@example.com", "hashedpwd", "", "", "", now, now)

	mock.ExpectQuery("SELECT .+ FROM users WHERE email").
		WithArgs("test@example.com").
		WillReturnRows(rows)

	user, err := repo.GetByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 1 {
		t.Fatalf("expected id 1, got %d", user.ID)
	}
	if user.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", user.Email)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_GetByEmail_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectQuery("SELECT .+ FROM users WHERE email").
		WithArgs("nonexistent@example.com").
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetByEmail(context.Background(), "nonexistent@example.com")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdateAvatarURL_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET avatar_url").
		WithArgs("new_avatar.png", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateAvatarURL(context.Background(), 1, "new_avatar.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdateAvatarURL_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET avatar_url").
		WithArgs("new_avatar.png", int64(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.UpdateAvatarURL(context.Background(), 999, "new_avatar.png")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdateProfile_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET username").
		WithArgs("newname", "newstatus", "newdesc", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	user := &domain.User{
		ID:          1,
		Username:    "newname",
		Status:      "newstatus",
		Description: "newdesc",
	}

	err = repo.UpdateProfile(context.Background(), user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdateProfile_UniqueViolation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET username").
		WithArgs("taken", "status", "desc", int64(1)).
		WillReturnError(&pq.Error{Code: "23505"})

	user := &domain.User{
		ID:          1,
		Username:    "taken",
		Status:      "status",
		Description: "desc",
	}

	err = repo.UpdateProfile(context.Background(), user)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdateProfile_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET username").
		WithArgs("newname", "status", "desc", int64(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	user := &domain.User{
		ID:          999,
		Username:    "newname",
		Status:      "status",
		Description: "desc",
	}

	err = repo.UpdateProfile(context.Background(), user)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdatePassword_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET password_hash").
		WithArgs("newhash", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdatePassword(context.Background(), 1, "newhash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdatePassword_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET password_hash").
		WithArgs("newhash", int64(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.UpdatePassword(context.Background(), 999, "newhash")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdateEmail_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET email").
		WithArgs("new@example.com", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateEmail(context.Background(), 1, "new@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdateEmail_UniqueViolation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET email").
		WithArgs("taken@example.com", int64(1)).
		WillReturnError(&pq.Error{Code: "23505"})

	err = repo.UpdateEmail(context.Background(), 1, "taken@example.com")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdateEmail_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET email").
		WithArgs("new@example.com", int64(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.UpdateEmail(context.Background(), 999, "new@example.com")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
