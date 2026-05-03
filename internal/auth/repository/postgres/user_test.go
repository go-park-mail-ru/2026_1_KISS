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
		"avatar_url", "status", "description", "is_verified", "is_admin",
		"plan", "last_active_at", "total_time_seconds",
		"created_at", "updated_at",
	}).AddRow(int64(1), "testuser", "test@example.com", "hashedpwd", "avatar.png", "active", "desc", true, false, "free", now, int64(0), now, now)

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
		"avatar_url", "status", "description", "is_verified", "is_admin",
		"plan", "last_active_at", "total_time_seconds",
		"created_at", "updated_at",
	}).AddRow(int64(1), "testuser", "test@example.com", "hashedpwd", "", "", "", true, false, "free", nil, int64(0), now, now)

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

func TestUserRepo_ListAll_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	now := time.Now()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(2))
	dataRows := sqlmock.NewRows([]string{
		"id", "username", "email", "password_hash",
		"avatar_url", "status", "description", "is_verified", "is_admin",
		"plan", "last_active_at", "total_time_seconds",
		"created_at", "updated_at",
	}).
		AddRow(int64(1), "user1", "user1@example.com", "hash1", "", "", "", true, false, "free", nil, int64(0), now, now).
		AddRow(int64(2), "user2", "user2@example.com", "hash2", "", "", "", true, false, "free", nil, int64(0), now, now)

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(countRows)
	mock.ExpectQuery("SELECT id, username").
		WithArgs(10, 0).
		WillReturnRows(dataRows)

	users, total, err := repo.ListAll(context.Background(), 10, 0, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 || len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_SetBanned_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET status").
		WithArgs("banned", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.SetBanned(context.Background(), 1, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_ListAll_WithSearch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	now := time.Now()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(1))
	dataRows := sqlmock.NewRows([]string{
		"id", "username", "email", "password_hash",
		"avatar_url", "status", "description", "is_verified", "is_admin",
		"plan", "last_active_at", "total_time_seconds",
		"created_at", "updated_at",
	}).
		AddRow(int64(1), "searchuser", "search@example.com", "hash1", "", "", "", true, false, "free", nil, int64(0), now, now)

	mock.ExpectQuery("SELECT COUNT").
		WithArgs("searchterm").
		WillReturnRows(countRows)
	mock.ExpectQuery("SELECT id, username").
		WithArgs("searchterm", 10, 0).
		WillReturnRows(dataRows)

	users, total, err := repo.ListAll(context.Background(), 10, 0, "searchterm", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_SetBanned_Unban(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET status").
		WithArgs("", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.SetBanned(context.Background(), 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_GetByUsername_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "username", "email", "password_hash",
		"avatar_url", "status", "description", "is_verified", "is_admin",
		"plan", "last_active_at", "total_time_seconds",
		"created_at", "updated_at",
	}).AddRow(int64(1), "testuser", "test@example.com", "hashedpwd", "", "", "", true, false, "free", nil, int64(0), now, now)

	mock.ExpectQuery("SELECT .+ FROM users WHERE username").
		WithArgs("testuser").
		WillReturnRows(rows)

	user, err := repo.GetByUsername(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 1 {
		t.Fatalf("expected id 1, got %d", user.ID)
	}
	if user.Username != "testuser" {
		t.Fatalf("expected username testuser, got %s", user.Username)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_GetByUsername_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectQuery("SELECT .+ FROM users WHERE username").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetByUsername(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_ListAll_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(0))
	dataRows := sqlmock.NewRows([]string{
		"id", "username", "email", "password_hash",
		"avatar_url", "status", "description", "is_verified", "is_admin",
		"plan", "last_active_at", "total_time_seconds",
		"created_at", "updated_at",
	})

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(countRows)
	mock.ExpectQuery("SELECT id, username").
		WithArgs(10, 0).
		WillReturnRows(dataRows)

	users, total, err := repo.ListAll(context.Background(), 10, 0, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 || len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_CountAll_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(int64(42))

	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(rows)

	count, err := repo.CountAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 42 {
		t.Fatalf("expected 42, got %d", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_CountAll_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectQuery("SELECT COUNT").
		WillReturnError(errors.New("db error"))

	_, err = repo.CountAll(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUserRepo_SetVerified_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET is_verified").
		WithArgs(true, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.SetVerified(context.Background(), 1, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_SetVerified_False(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET is_verified").
		WithArgs(false, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.SetVerified(context.Background(), 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_SetVerified_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET is_verified").
		WithArgs(true, int64(1)).
		WillReturnError(errors.New("db error"))

	err = repo.SetVerified(context.Background(), 1, true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUserRepo_AdminUpdateUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET username").
		WithArgs("newname", "new@example.com", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.AdminUpdateUser(context.Background(), 1, "newname", "new@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_AdminUpdateUser_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET username").
		WithArgs("newname", "new@example.com", int64(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.AdminUpdateUser(context.Background(), 999, "newname", "new@example.com")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_AdminUpdateUser_UniqueViolation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET username").
		WithArgs("taken", "taken@example.com", int64(1)).
		WillReturnError(&pq.Error{Code: "23505"})

	err = repo.AdminUpdateUser(context.Background(), 1, "taken", "taken@example.com")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdatePlan_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET plan").
		WithArgs(domain.PlanAdmin, true, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdatePlan(context.Background(), 1, domain.PlanAdmin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdatePlan_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET plan").
		WithArgs(domain.PlanPro, false, int64(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.UpdatePlan(context.Background(), 999, domain.PlanPro)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdatePlan_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET plan").
		WithArgs(domain.PlanFree, false, int64(1)).
		WillReturnError(errors.New("db error"))

	err = repo.UpdatePlan(context.Background(), 1, domain.PlanFree)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUserRepo_UpdateLastActive_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	now := time.Now()

	mock.ExpectExec("UPDATE users SET last_active_at").
		WithArgs(now, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateLastActive(context.Background(), 1, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_UpdateLastActive_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	now := time.Now()

	mock.ExpectExec("UPDATE users SET last_active_at").
		WithArgs(now, int64(1)).
		WillReturnError(errors.New("db error"))

	err = repo.UpdateLastActive(context.Background(), 1, now)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUserRepo_IncrementTotalTime_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET total_time_seconds").
		WithArgs(int64(3600), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.IncrementTotalTime(context.Background(), 1, 3600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_IncrementTotalTime_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("UPDATE users SET total_time_seconds").
		WithArgs(int64(100), int64(1)).
		WillReturnError(errors.New("db error"))

	err = repo.IncrementTotalTime(context.Background(), 1, 100)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUserRepo_DeleteUnverifiedBefore_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)
	cutoff := time.Now().Add(-24 * time.Hour)

	mock.ExpectExec("DELETE FROM users WHERE is_verified = FALSE AND created_at").
		WithArgs(cutoff).
		WillReturnResult(sqlmock.NewResult(0, 5))

	count, err := repo.DeleteUnverifiedBefore(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5 deleted, got %d", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserRepo_DeleteUnverifiedBefore_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewUserRepository(db)
	cutoff := time.Now().Add(-24 * time.Hour)

	mock.ExpectExec("DELETE FROM users WHERE is_verified = FALSE AND created_at").
		WithArgs(cutoff).
		WillReturnError(errors.New("connection refused"))

	_, err = repo.DeleteUnverifiedBefore(context.Background(), cutoff)
	if err == nil {
		t.Fatal("expected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
