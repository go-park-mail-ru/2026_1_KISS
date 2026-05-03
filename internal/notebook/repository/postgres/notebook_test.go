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

func TestNotebookRepo_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewNotebookRepository(db)
		now := time.Now()

		mock.ExpectQuery(`INSERT INTO notebooks`).
			WithArgs(int64(1), "My Notebook").
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(int64(10), now, now))

		nb := &domain.Notebook{OwnerID: 1, Title: "My Notebook"}
		id, err := repo.Create(context.Background(), nb)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if id != 10 {
			t.Fatalf("expected id 10, got %d", id)
		}
		if nb.CreatedAt.IsZero() {
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

		repo := NewNotebookRepository(db)

		mock.ExpectQuery(`INSERT INTO notebooks`).
			WithArgs(int64(1), "bad").
			WillReturnError(fmt.Errorf("db error"))

		nb := &domain.Notebook{OwnerID: 1, Title: "bad"}
		_, err = repo.Create(context.Background(), nb)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestNotebookRepo_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewNotebookRepository(db)
		now := time.Now()

		mock.ExpectQuery(`SELECT id, owner_id, title, is_public, created_at, updated_at FROM notebooks WHERE id`).
			WithArgs(int64(5)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "title", "is_public", "created_at", "updated_at"}).
				AddRow(int64(5), int64(1), "Test", false, now, now))

		nb, err := repo.GetByID(context.Background(), 5)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if nb.ID != 5 {
			t.Fatalf("expected ID=5, got %d", nb.ID)
		}
		if nb.Title != "Test" {
			t.Fatalf("expected title 'Test', got %q", nb.Title)
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

		repo := NewNotebookRepository(db)

		mock.ExpectQuery(`SELECT id, owner_id, title, is_public, created_at, updated_at FROM notebooks WHERE id`).
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

func TestNotebookRepo_GetByOwnerID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewNotebookRepository(db)
		now := time.Now()

		mock.ExpectQuery(`SELECT id, owner_id, title, is_public, created_at, updated_at`).
			WithArgs(int64(1), "", 10, 0).
			WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "title", "is_public", "created_at", "updated_at"}).
				AddRow(int64(1), int64(1), "NB1", false, now, now).
				AddRow(int64(2), int64(1), "NB2", true, now, now))

		nbs, err := repo.GetByOwnerID(context.Background(), 1, 10, 0, "")
		if err != nil {
			t.Fatalf("GetByOwnerID() error = %v", err)
		}
		if len(nbs) != 2 {
			t.Fatalf("expected 2 notebooks, got %d", len(nbs))
		}
		if nbs[0].Title != "NB1" {
			t.Fatalf("expected first title 'NB1', got %q", nbs[0].Title)
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

		repo := NewNotebookRepository(db)

		mock.ExpectQuery(`SELECT id, owner_id, title, is_public, created_at, updated_at`).
			WithArgs(int64(99), "", 10, 0).
			WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "title", "is_public", "created_at", "updated_at"}))

		nbs, err := repo.GetByOwnerID(context.Background(), 99, 10, 0, "")
		if err != nil {
			t.Fatalf("GetByOwnerID() error = %v", err)
		}
		if len(nbs) != 0 {
			t.Fatalf("expected 0 notebooks, got %d", len(nbs))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestNotebookRepo_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewNotebookRepository(db)

		mock.ExpectExec(`DELETE FROM notebooks WHERE id`).
			WithArgs(int64(5)).
			WillReturnResult(sqlmock.NewResult(0, 1))

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

		repo := NewNotebookRepository(db)

		mock.ExpectExec(`DELETE FROM notebooks WHERE id`).
			WithArgs(int64(999)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err = repo.Delete(context.Background(), 999)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestNotebookRepo_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewNotebookRepository(db)
		now := time.Now()

		mock.ExpectQuery(`UPDATE notebooks SET`).
			WithArgs("Updated Title", true, int64(5), int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(now))

		nb := &domain.Notebook{ID: 5, OwnerID: 1, Title: "Updated Title", IsPublic: true}
		err = repo.Update(context.Background(), nb)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if nb.UpdatedAt.IsZero() {
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

		repo := NewNotebookRepository(db)

		mock.ExpectQuery(`UPDATE notebooks SET`).
			WithArgs("Title", false, int64(999), int64(1)).
			WillReturnError(sql.ErrNoRows)

		nb := &domain.Notebook{ID: 999, OwnerID: 1, Title: "Title"}
		err = repo.Update(context.Background(), nb)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestNotebookRepo_CountByOwnerID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewNotebookRepository(db)

		mock.ExpectQuery(`SELECT COUNT`).
			WithArgs(int64(1), "").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

		count, err := repo.CountByOwnerID(context.Background(), 1, "")
		if err != nil {
			t.Fatalf("CountByOwnerID() error = %v", err)
		}
		if count != 42 {
			t.Fatalf("expected count 42, got %d", count)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestListAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create mock: %v", err)
		}
		defer db.Close()

		repo := NewNotebookRepository(db)
		now := time.Now()

		rows := sqlmock.NewRows([]string{"id", "owner_id", "title", "is_public", "created_at", "updated_at"}).
			AddRow(int64(1), int64(1), "Notebook 1", true, now, now)

		mock.ExpectQuery(`SELECT id, owner_id, title, is_public, created_at, updated_at`).
			WithArgs("", 10, 0).
			WillReturnRows(rows)

		notebooks, err := repo.ListAll(context.Background(), 10, 0, "")
		if err != nil {
			t.Fatalf("ListAll() error = %v", err)
		}
		if len(notebooks) != 1 {
			t.Fatalf("expected 1 notebook, got %d", len(notebooks))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestCountAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create mock: %v", err)
		}
		defer db.Close()

		repo := NewNotebookRepository(db)

		mock.ExpectQuery(`SELECT COUNT`).
			WithArgs("").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

		count, err := repo.CountAll(context.Background(), "")
		if err != nil {
			t.Fatalf("CountAll() error = %v", err)
		}
		if count != 42 {
			t.Fatalf("expected count 42, got %d", count)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestGetByOwnerID_WithSearch(t *testing.T) {
	t.Run("success with search", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewNotebookRepository(db)
		now := time.Now()

		mock.ExpectQuery(`SELECT id, owner_id, title, is_public, created_at, updated_at`).
			WithArgs(int64(1), "test", 10, 0).
			WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "title", "is_public", "created_at", "updated_at"}).
				AddRow(int64(1), int64(1), "test notebook", false, now, now))

		nbs, err := repo.GetByOwnerID(context.Background(), 1, 10, 0, "test")
		if err != nil {
			t.Fatalf("GetByOwnerID() error = %v", err)
		}
		if len(nbs) != 1 {
			t.Fatalf("expected 1 notebook, got %d", len(nbs))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}
