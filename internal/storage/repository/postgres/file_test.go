package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestFileRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO files`).
		WithArgs(int64(1), (*int64)(nil), "files", "test.txt", "files/uuid.txt", "/uploads/files/uuid.txt", "text/plain", int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("file-uuid-1", now))

	file := &domain.File{
		OwnerID:    1,
		Category:   domain.FileCategoryGeneral,
		Filename:   "test.txt",
		StorageKey: "files/uuid.txt",
		URL:        "/uploads/files/uuid.txt",
		MIMEType:   "text/plain",
		Size:       100,
	}

	err = repo.Create(context.Background(), file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.ID != "file-uuid-1" {
		t.Errorf("expected id file-uuid-1, got %s", file.ID)
	}
}

func TestFileRepo_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)

	mock.ExpectQuery(`SELECT .+ FROM files WHERE id`).
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetByID(context.Background(), "nonexistent")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFileRepo_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)
	now := time.Now()

	mock.ExpectQuery(`SELECT .+ FROM files WHERE id`).
		WithArgs("file-1").
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "owner_id", "notebook_id", "category", "filename", "storage_key", "url", "mime_type", "size", "created_at", "is_public", "share_token", "share_expires_at", "downloads_count"},
		).AddRow("file-1", int64(1), nil, "datasets", "data.csv", "datasets/uuid.csv", "/uploads/datasets/uuid.csv", "text/csv", int64(500), now, false, nil, nil, int64(0)))

	file, err := repo.GetByID(context.Background(), "file-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.ID != "file-1" {
		t.Errorf("expected id file-1, got %s", file.ID)
	}
	if file.Category != domain.FileCategoryDataset {
		t.Errorf("expected category datasets, got %s", file.Category)
	}
}

func TestFileRepo_Delete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)

	mock.ExpectExec(`DELETE FROM files WHERE id`).
		WithArgs("nonexistent").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Delete(context.Background(), "nonexistent")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFileRepo_Delete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)

	mock.ExpectExec(`DELETE FROM files WHERE id`).
		WithArgs("file-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(context.Background(), "file-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileRepo_ListByOwner_WithCategory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)
	now := time.Now()

	mock.ExpectQuery(`SELECT COUNT`).
		WithArgs(int64(1), "datasets").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT .+ FROM files WHERE owner_id`).
		WithArgs(int64(1), "datasets", 20, 0).
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "owner_id", "notebook_id", "category", "filename", "storage_key", "url", "mime_type", "size", "created_at", "is_public", "share_token", "share_expires_at", "downloads_count"},
		).AddRow("f-1", int64(1), nil, "datasets", "data.csv", "datasets/uuid.csv", "/uploads/datasets/uuid.csv", "text/csv", int64(500), now, false, nil, nil, int64(0)))

	files, total, err := repo.ListByOwner(context.Background(), 1, "datasets", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Category != domain.FileCategoryDataset {
		t.Errorf("expected datasets, got %s", files[0].Category)
	}
}

func TestFileRepo_ListByOwner_AllCategories(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)

	mock.ExpectQuery(`SELECT COUNT`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`SELECT .+ FROM files WHERE owner_id`).
		WithArgs(int64(1), 20, 0).
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "owner_id", "notebook_id", "category", "filename", "storage_key", "url", "mime_type", "size", "created_at", "is_public", "share_token", "share_expires_at", "downloads_count"},
		))

	files, total, err := repo.ListByOwner(context.Background(), 1, "", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestFileRepo_ListAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)
	now := time.Now()

	mock.ExpectQuery(`SELECT COUNT`).
		WithArgs("datasets", int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	mock.ExpectQuery(`SELECT .+ FROM files WHERE`).
		WithArgs("datasets", int64(1), 10, 0).
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "owner_id", "notebook_id", "category", "filename", "storage_key", "url", "mime_type", "size", "created_at", "is_public", "share_token", "share_expires_at", "downloads_count"},
		).
			AddRow("f-1", int64(1), nil, "datasets", "a.csv", "datasets/a.csv", "/uploads/datasets/a.csv", "text/csv", int64(100), now, false, nil, nil, int64(0)).
			AddRow("f-2", int64(1), nil, "datasets", "b.csv", "datasets/b.csv", "/uploads/datasets/b.csv", "text/csv", int64(200), now, false, nil, nil, int64(0)))

	files, total, err := repo.ListAll(context.Background(), "datasets", 1, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestFileRepo_ListAll_NoFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)

	mock.ExpectQuery(`SELECT COUNT`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`SELECT .+ FROM files WHERE`).
		WithArgs(10, 0).
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "owner_id", "notebook_id", "category", "filename", "storage_key", "url", "mime_type", "size", "created_at", "is_public", "share_token", "share_expires_at", "downloads_count"},
		))

	files, total, err := repo.ListAll(context.Background(), "", 0, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestFileRepo_GetStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewFileRepository(db)

	mock.ExpectQuery(`SELECT category, COUNT`).
		WillReturnRows(sqlmock.NewRows([]string{"category", "count", "sum"}).
			AddRow("datasets", int64(5), int64(1024)).
			AddRow("files", int64(3), int64(512)))

	stats, err := repo.GetStats(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalFiles != 8 {
		t.Errorf("expected 8 total files, got %d", stats.TotalFiles)
	}
	if stats.TotalSizeBytes != 1536 {
		t.Errorf("expected 1536 bytes, got %d", stats.TotalSizeBytes)
	}
}
