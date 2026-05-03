package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestPermissionRepo_Upsert(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewPermissionRepository(db)

		mock.ExpectExec(`INSERT INTO file_permissions`).
			WithArgs(int64(1), int64(2), "editor").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err = repo.Upsert(context.Background(), &domain.FilePermission{
			NotebookID:      1,
			UserID:          2,
			PermissionLevel: "editor",
		})
		if err != nil {
			t.Fatalf("Upsert() error = %v", err)
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

		repo := NewPermissionRepository(db)

		mock.ExpectExec(`INSERT INTO file_permissions`).
			WithArgs(int64(1), int64(2), "editor").
			WillReturnError(fmt.Errorf("db error"))

		err = repo.Upsert(context.Background(), &domain.FilePermission{
			NotebookID:      1,
			UserID:          2,
			PermissionLevel: "editor",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestPermissionRepo_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewPermissionRepository(db)

		mock.ExpectExec(`DELETE FROM file_permissions`).
			WithArgs(int64(1), int64(2)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err = repo.Delete(context.Background(), 1, 2)
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

		repo := NewPermissionRepository(db)

		mock.ExpectExec(`DELETE FROM file_permissions`).
			WithArgs(int64(1), int64(2)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err = repo.Delete(context.Background(), 1, 2)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("db_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewPermissionRepository(db)

		mock.ExpectExec(`DELETE FROM file_permissions`).
			WithArgs(int64(1), int64(2)).
			WillReturnError(fmt.Errorf("db error"))

		err = repo.Delete(context.Background(), 1, 2)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("rows_affected_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewPermissionRepository(db)

		mock.ExpectExec(`DELETE FROM file_permissions`).
			WithArgs(int64(1), int64(2)).
			WillReturnResult(sqlmock.NewErrorResult(fmt.Errorf("rows affected error")))

		err = repo.Delete(context.Background(), 1, 2)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestPermissionRepo_GetByNotebookID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewPermissionRepository(db)

		mock.ExpectQuery(`SELECT notebook_id, user_id, permission_level FROM file_permissions`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"notebook_id", "user_id", "permission_level"}).
				AddRow(int64(1), int64(2), "editor").
				AddRow(int64(1), int64(3), "readonly"))

		perms, err := repo.GetByNotebookID(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetByNotebookID() error = %v", err)
		}
		if len(perms) != 2 {
			t.Fatalf("expected 2 permissions, got %d", len(perms))
		}
		if perms[0].UserID != 2 || perms[0].PermissionLevel != "editor" {
			t.Errorf("unexpected first permission: %+v", perms[0])
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

		repo := NewPermissionRepository(db)

		mock.ExpectQuery(`SELECT notebook_id, user_id, permission_level FROM file_permissions`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"notebook_id", "user_id", "permission_level"}))

		perms, err := repo.GetByNotebookID(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetByNotebookID() error = %v", err)
		}
		if len(perms) != 0 {
			t.Fatalf("expected 0 permissions, got %d", len(perms))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("db_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewPermissionRepository(db)

		mock.ExpectQuery(`SELECT notebook_id, user_id, permission_level FROM file_permissions`).
			WithArgs(int64(1)).
			WillReturnError(fmt.Errorf("db error"))

		_, err = repo.GetByNotebookID(context.Background(), 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestPermissionRepo_GetPermission(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewPermissionRepository(db)

		mock.ExpectQuery(`SELECT notebook_id, user_id, permission_level FROM file_permissions`).
			WithArgs(int64(1), int64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"notebook_id", "user_id", "permission_level"}).
				AddRow(int64(1), int64(2), "editor"))

		perm, err := repo.GetPermission(context.Background(), 1, 2)
		if err != nil {
			t.Fatalf("GetPermission() error = %v", err)
		}
		if perm.PermissionLevel != "editor" {
			t.Errorf("expected editor, got %s", perm.PermissionLevel)
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

		repo := NewPermissionRepository(db)

		mock.ExpectQuery(`SELECT notebook_id, user_id, permission_level FROM file_permissions`).
			WithArgs(int64(1), int64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"notebook_id", "user_id", "permission_level"}))

		_, err = repo.GetPermission(context.Background(), 1, 2)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("db_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewPermissionRepository(db)

		mock.ExpectQuery(`SELECT notebook_id, user_id, permission_level FROM file_permissions`).
			WithArgs(int64(1), int64(2)).
			WillReturnError(fmt.Errorf("db error"))

		_, err = repo.GetPermission(context.Background(), 1, 2)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}
