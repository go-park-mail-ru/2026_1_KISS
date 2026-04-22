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

func TestBlockRepo_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)
		now := time.Now()

		mock.ExpectQuery(`INSERT INTO blocks`).
			WithArgs(int64(1), "code", "python", "print('hi')", 0).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(int64(7), now, now))

		b := &domain.Block{
			NotebookID: 1,
			Type:       "code",
			Language:   "python",
			Content:    "print('hi')",
			Position:   0,
		}
		id, err := repo.Create(context.Background(), b)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if id != 7 {
			t.Fatalf("expected id 7, got %d", id)
		}
		if b.CreatedAt.IsZero() {
			t.Fatal("CreatedAt should be set")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectQuery(`INSERT INTO blocks`).
			WithArgs(int64(1), "code", "go", "", 0).
			WillReturnError(fmt.Errorf("unique violation"))

		b := &domain.Block{
			NotebookID: 1,
			Type:       "code",
			Language:   "go",
			Position:   0,
		}
		_, err = repo.Create(context.Background(), b)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestBlockRepo_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)
		now := time.Now()
		execCount := 3

		mock.ExpectQuery(`SELECT id, notebook_id, type, language, content, position, execution_count, created_at, updated_at FROM blocks WHERE id`).
			WithArgs(int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "notebook_id", "type", "language", "content", "position", "execution_count", "created_at", "updated_at"}).
				AddRow(int64(7), int64(1), "code", "python", "x=1", 0, execCount, now, now))

		b, err := repo.GetByID(context.Background(), 7)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if b.ID != 7 {
			t.Fatalf("expected ID=7, got %d", b.ID)
		}
		if b.Content != "x=1" {
			t.Fatalf("expected content 'x=1', got %q", b.Content)
		}
		if b.Outputs == nil {
			t.Fatal("Outputs should be initialized (not nil)")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectQuery(`SELECT id, notebook_id, type, language, content, position, execution_count, created_at, updated_at FROM blocks WHERE id`).
			WithArgs(int64(999)).
			WillReturnError(sql.ErrNoRows)

		_, err = repo.GetByID(context.Background(), 999)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestBlockRepo_GetByNotebookID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)
		now := time.Now()

		mock.ExpectQuery(`SELECT id, notebook_id, type, language, content, position, execution_count, created_at, updated_at FROM blocks WHERE notebook_id`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "notebook_id", "type", "language", "content", "position", "execution_count", "created_at", "updated_at"}).
				AddRow(int64(1), int64(1), "code", "python", "a=1", 0, nil, now, now).
				AddRow(int64(2), int64(1), "markdown", "", "# Title", 1, nil, now, now))

		blocks, err := repo.GetByNotebookID(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetByNotebookID() error = %v", err)
		}
		if len(blocks) != 2 {
			t.Fatalf("expected 2 blocks, got %d", len(blocks))
		}
		if blocks[0].Type != "code" {
			t.Fatalf("expected first block type 'code', got %q", blocks[0].Type)
		}
		if blocks[1].Outputs == nil {
			t.Fatal("Outputs should be initialized (not nil)")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("empty", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectQuery(`SELECT id, notebook_id, type, language, content, position, execution_count, created_at, updated_at FROM blocks WHERE notebook_id`).
			WithArgs(int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "notebook_id", "type", "language", "content", "position", "execution_count", "created_at", "updated_at"}))

		blocks, err := repo.GetByNotebookID(context.Background(), 99)
		if err != nil {
			t.Fatalf("GetByNotebookID() error = %v", err)
		}
		if len(blocks) != 0 {
			t.Fatalf("expected 0 blocks, got %d", len(blocks))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestBlockRepo_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)
		now := time.Now()

		mock.ExpectQuery(`UPDATE blocks SET`).
			WithArgs("new content", "code", "go", int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(now))

		b := &domain.Block{ID: 7, Content: "new content", Type: "code", Language: "go"}
		err = repo.Update(context.Background(), b)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if b.UpdatedAt.IsZero() {
			t.Fatal("UpdatedAt should be set")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectQuery(`UPDATE blocks SET`).
			WithArgs("content", "code", "python", int64(999)).
			WillReturnError(sql.ErrNoRows)

		b := &domain.Block{ID: 999, Content: "content", Type: "code", Language: "python"}
		err = repo.Update(context.Background(), b)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestBlockRepo_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT position, notebook_id FROM blocks WHERE id`).
			WithArgs(int64(5)).
			WillReturnRows(sqlmock.NewRows([]string{"position", "notebook_id"}).
				AddRow(2, int64(1)))
		mock.ExpectExec(`DELETE FROM blocks WHERE id`).
			WithArgs(int64(5)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE blocks SET position`).
			WithArgs(int64(1), 2).
			WillReturnResult(sqlmock.NewResult(0, 3))
		mock.ExpectCommit()

		err = repo.Delete(context.Background(), 5)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT position, notebook_id FROM blocks WHERE id`).
			WithArgs(int64(999)).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		err = repo.Delete(context.Background(), 999)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}
