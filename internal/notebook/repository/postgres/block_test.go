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

		mock.ExpectQuery(`SELECT id, block_id, position, output_type, content, created_at FROM block_outputs`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "block_id", "position", "output_type", "content", "created_at"}))

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

	t.Run("output_load_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)
		now := time.Now()

		mock.ExpectQuery(`SELECT id, notebook_id, type, language, content, position, execution_count, created_at, updated_at FROM blocks WHERE id`).
			WithArgs(int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "notebook_id", "type", "language", "content", "position", "execution_count", "created_at", "updated_at"}).
				AddRow(int64(7), int64(1), "code", "python", "x=1", 0, nil, now, now))

		mock.ExpectQuery(`SELECT id, block_id, position, output_type, content, created_at FROM block_outputs`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(fmt.Errorf("output query error"))

		_, err = repo.GetByID(context.Background(), 7)
		if err == nil {
			t.Fatal("expected error, got nil")
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

		mock.ExpectQuery(`SELECT id, block_id, position, output_type, content, created_at FROM block_outputs`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "block_id", "position", "output_type", "content", "created_at"}))

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

	t.Run("with_outputs", func(t *testing.T) {
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
				AddRow(int64(1), int64(1), "code", "python", "print(42)", 0, nil, now, now))

		mock.ExpectQuery(`SELECT id, block_id, position, output_type, content, created_at FROM block_outputs`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "block_id", "position", "output_type", "content", "created_at"}).
				AddRow(int64(10), int64(1), 0, "stdout", "42", now))

		blocks, err := repo.GetByNotebookID(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetByNotebookID() error = %v", err)
		}
		if len(blocks) != 1 {
			t.Fatalf("expected 1 block, got %d", len(blocks))
		}
		if len(blocks[0].Outputs) != 1 {
			t.Fatalf("expected 1 output, got %d", len(blocks[0].Outputs))
		}
		if blocks[0].Outputs[0].Content != "42" {
			t.Fatalf("expected output '42', got %q", blocks[0].Outputs[0].Content)
		}
	})

	t.Run("output_load_error", func(t *testing.T) {
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
				AddRow(int64(1), int64(1), "code", "python", "a=1", 0, nil, now, now))

		mock.ExpectQuery(`SELECT id, block_id, position, output_type, content, created_at FROM block_outputs`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(fmt.Errorf("output query error"))

		_, err = repo.GetByNotebookID(context.Background(), 1)
		if err == nil {
			t.Fatal("expected error, got nil")
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

func TestBlockRepo_SaveOutputs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM block_outputs WHERE block_id`).
			WithArgs(int64(7)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(`INSERT INTO block_outputs`).
			WithArgs(int64(7), 0, "stdout", "hello", int64(7), 1, "stderr", "warn").
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		outputs := []domain.BlockOutput{
			{Position: 0, OutputType: "stdout", Content: "hello"},
			{Position: 1, OutputType: "stderr", Content: "warn"},
		}
		err = repo.SaveOutputs(context.Background(), 7, outputs)
		if err != nil {
			t.Fatalf("SaveOutputs() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("empty_outputs", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM block_outputs WHERE block_id`).
			WithArgs(int64(7)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		err = repo.SaveOutputs(context.Background(), 7, nil)
		if err != nil {
			t.Fatalf("SaveOutputs() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("begin_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin().WillReturnError(fmt.Errorf("begin error"))

		err = repo.SaveOutputs(context.Background(), 7, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("delete_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM block_outputs WHERE block_id`).
			WithArgs(int64(7)).
			WillReturnError(fmt.Errorf("delete error"))
		mock.ExpectRollback()

		err = repo.SaveOutputs(context.Background(), 7, []domain.BlockOutput{{Position: 0, OutputType: "stdout", Content: "hi"}})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("commit_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM block_outputs WHERE block_id`).
			WithArgs(int64(7)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(`INSERT INTO block_outputs`).
			WithArgs(int64(7), 0, "stdout", "hi").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(fmt.Errorf("commit error"))

		err = repo.SaveOutputs(context.Background(), 7, []domain.BlockOutput{{Position: 0, OutputType: "stdout", Content: "hi"}})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("insert_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM block_outputs WHERE block_id`).
			WithArgs(int64(7)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(`INSERT INTO block_outputs`).
			WithArgs(int64(7), 0, "stdout", "hi").
			WillReturnError(fmt.Errorf("insert error"))
		mock.ExpectRollback()

		err = repo.SaveOutputs(context.Background(), 7, []domain.BlockOutput{{Position: 0, OutputType: "stdout", Content: "hi"}})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestBlockRepo_GetOutputsByBlockIDs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)
		now := time.Now()

		mock.ExpectQuery(`SELECT id, block_id, position, output_type, content, created_at FROM block_outputs`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "block_id", "position", "output_type", "content", "created_at"}).
				AddRow(int64(1), int64(7), 0, "stdout", "hello", now).
				AddRow(int64(2), int64(8), 0, "result", "42", now))

		result, err := repo.GetOutputsByBlockIDs(context.Background(), []int64{7, 8})
		if err != nil {
			t.Fatalf("GetOutputsByBlockIDs() error = %v", err)
		}
		if len(result[7]) != 1 {
			t.Fatalf("expected 1 output for block 7, got %d", len(result[7]))
		}
		if result[7][0].Content != "hello" {
			t.Fatalf("expected content 'hello', got %q", result[7][0].Content)
		}
		if len(result[8]) != 1 {
			t.Fatalf("expected 1 output for block 8, got %d", len(result[8]))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("empty_ids", func(t *testing.T) {
		db, _, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)
		result, err := repo.GetOutputsByBlockIDs(context.Background(), nil)
		if err != nil {
			t.Fatalf("GetOutputsByBlockIDs() error = %v", err)
		}
		if len(result) != 0 {
			t.Fatalf("expected empty map, got %d entries", len(result))
		}
	})

	t.Run("query_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectQuery(`SELECT id, block_id, position, output_type, content, created_at FROM block_outputs`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(fmt.Errorf("query error"))

		_, err = repo.GetOutputsByBlockIDs(context.Background(), []int64{7})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestBlockRepo_ReorderBlocks(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE blocks SET position = -`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`UPDATE blocks SET position`).
			WithArgs(0, int64(3), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE blocks SET position`).
			WithArgs(1, int64(2), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err = repo.ReorderBlocks(context.Background(), 1, []int64{3, 2})
		if err != nil {
			t.Fatalf("ReorderBlocks() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("begin_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin().WillReturnError(fmt.Errorf("begin error"))

		err = repo.ReorderBlocks(context.Background(), 1, []int64{3, 2})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("negate_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE blocks SET position = -`).
			WithArgs(int64(1)).
			WillReturnError(fmt.Errorf("negate error"))
		mock.ExpectRollback()

		err = repo.ReorderBlocks(context.Background(), 1, []int64{3, 2})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("commit_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE blocks SET position = -`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`UPDATE blocks SET position`).
			WithArgs(0, int64(3), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(fmt.Errorf("commit error"))

		err = repo.ReorderBlocks(context.Background(), 1, []int64{3})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("set_position_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE blocks SET position = -`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`UPDATE blocks SET position`).
			WithArgs(0, int64(3), int64(1)).
			WillReturnError(fmt.Errorf("set position error"))
		mock.ExpectRollback()

		err = repo.ReorderBlocks(context.Background(), 1, []int64{3, 2})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestBlockRepo_CreateBatch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO blocks`).
			WithArgs(int64(1), "code", "python", "print('hi')", 0).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))
		mock.ExpectQuery(`INSERT INTO blocks`).
			WithArgs(int64(1), "text", "markdown", "# hello", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(11)))
		mock.ExpectCommit()

		blocks := []domain.Block{
			{NotebookID: 1, Type: "code", Language: "python", Content: "print('hi')", Position: 0},
			{NotebookID: 1, Type: "text", Language: "markdown", Content: "# hello", Position: 1},
		}

		ids, err := repo.CreateBatch(context.Background(), blocks)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 2 || ids[0] != 10 || ids[1] != 11 {
			t.Errorf("want ids [10 11], got %v", ids)
		}
	})

	t.Run("insert_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO blocks`).
			WithArgs(int64(1), "code", "python", "", 0).
			WillReturnError(fmt.Errorf("insert error"))
		mock.ExpectRollback()

		blocks := []domain.Block{
			{NotebookID: 1, Type: "code", Language: "python", Content: "", Position: 0},
		}

		_, err = repo.CreateBatch(context.Background(), blocks)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("begin_error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := NewBlockRepository(db)

		mock.ExpectBegin().WillReturnError(fmt.Errorf("begin error"))

		_, err = repo.CreateBatch(context.Background(), []domain.Block{{NotebookID: 1}})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
