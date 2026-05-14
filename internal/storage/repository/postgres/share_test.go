package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestFileShareRepo_Upsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileShareRepository(db)
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO file_shares`).
		WithArgs("file-1", int64(2), "download").
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(now))

	share := &domain.FileShare{FileID: "file-1", UserID: 2, Level: "download"}
	if err := repo.Upsert(context.Background(), share); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if share.CreatedAt.IsZero() {
		t.Errorf("expected created_at to be set")
	}
}

func TestFileShareRepo_Delete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileShareRepository(db)

	mock.ExpectExec(`DELETE FROM file_shares`).
		WithArgs("file-1", int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.Delete(context.Background(), "file-1", 2); err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFileShareRepo_Delete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileShareRepository(db)

	mock.ExpectExec(`DELETE FROM file_shares`).
		WithArgs("file-1", int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.Delete(context.Background(), "file-1", 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileShareRepo_GetByFileID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileShareRepository(db)
	now := time.Now()

	mock.ExpectQuery(`SELECT .+ FROM file_shares`).
		WithArgs("file-1").
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "email", "permission_level", "created_at"}).
			AddRow("file-1", int64(2), "a@b.com", "view", now).
			AddRow("file-1", int64(3), "c@d.com", "download", now))

	shares, err := repo.GetByFileID(context.Background(), "file-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shares) != 2 {
		t.Fatalf("expected 2 shares, got %d", len(shares))
	}
	if shares[0].Email != "a@b.com" {
		t.Errorf("expected first email a@b.com, got %s", shares[0].Email)
	}
}

func TestFileShareRepo_GetPermission_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileShareRepository(db)

	mock.ExpectQuery(`SELECT .+ FROM file_shares WHERE file_id`).
		WithArgs("file-1", int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "permission_level", "created_at"}))

	_, err = repo.GetPermission(context.Background(), "file-1", 2)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFileShareRepo_ListByUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileShareRepository(db)
	now := time.Now()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM file_shares`).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT .+ FROM file_shares fs`).
		WithArgs(int64(2), 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "owner_id", "notebook_id", "category", "filename", "storage_key", "url",
			"mime_type", "size", "created_at", "is_public", "share_token", "share_expires_at",
			"downloads_count", "permission_level",
		}).AddRow("f-1", int64(1), nil, "files", "doc.txt", "files/uuid.txt", "/uploads/files/uuid.txt",
			"text/plain", int64(123), now, false, nil, nil, int64(0), "download"))

	files, total, err := repo.ListByUserID(context.Background(), 2, 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].YourPermission != "download" {
		t.Errorf("expected permission download, got %s", files[0].YourPermission)
	}
}

func TestFileRepo_SetPublic_Enable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)
	token := "token-1"

	mock.ExpectExec(`UPDATE files SET is_public`).
		WithArgs(true, &token, (*time.Time)(nil), "file-1", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.SetPublic(context.Background(), "file-1", 1, true, &token, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileRepo_SetPublic_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)

	mock.ExpectExec(`UPDATE files SET is_public`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.SetPublic(context.Background(), "file-1", 1, false, nil, nil); err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFileRepo_GetByShareToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)
	now := time.Now()
	token := "token-xyz"

	mock.ExpectQuery(`SELECT .+ FROM files WHERE share_token`).
		WithArgs("token-xyz").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "owner_id", "notebook_id", "category", "filename", "storage_key", "url",
			"mime_type", "size", "created_at", "is_public", "share_token", "share_expires_at", "downloads_count",
		}).AddRow("file-1", int64(1), nil, "files", "doc.txt", "files/uuid.txt", "/uploads/files/uuid.txt",
			"text/plain", int64(123), now, true, token, nil, int64(0)))

	file, err := repo.GetByShareToken(context.Background(), "token-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !file.IsPublic {
		t.Errorf("expected file to be public")
	}
	if file.ShareToken == nil || *file.ShareToken != token {
		t.Errorf("expected share token %q, got %v", token, file.ShareToken)
	}
}

func TestFileRepo_IncrementDownloads(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)

	mock.ExpectExec(`UPDATE files SET downloads_count`).
		WithArgs("file-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.IncrementDownloads(context.Background(), "file-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileRepo_Rename(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)

	mock.ExpectExec(`UPDATE files SET filename`).
		WithArgs("new.txt", "file-1", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.Rename(context.Background(), "file-1", 1, "new.txt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileRepo_Rename_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)

	mock.ExpectExec(`UPDATE files SET filename`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.Rename(context.Background(), "file-1", 2, "new.txt"); err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
