package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestCommentRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)
	comment := &domain.Comment{UserID: 1, BlockID: 2, Text: "hello"}

	mock.ExpectQuery(`WITH inserted AS`).
		WithArgs(int64(1), int64(2), "hello").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "username", "block_id", "text", "created_at"}).
			AddRow(10, 1, "alice", 2, "hello", time.Now()))

	id, err := repo.Create(context.Background(), comment)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), id)
	assert.Equal(t, "alice", comment.Username)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCommentRepo_Create_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	mock.ExpectQuery(`WITH inserted AS`).
		WithArgs(int64(1), int64(2), "hello").
		WillReturnError(assert.AnError)

	_, err = repo.Create(context.Background(), &domain.Comment{UserID: 1, BlockID: 2, Text: "hello"})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCommentRepo_GetByBlockID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	now := time.Now()
	mock.ExpectQuery(`SELECT.*FROM comments c.*JOIN users u.*WHERE c.block_id`).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "username", "block_id", "text", "created_at"}).
			AddRow(1, 1, "alice", 2, "first", now).
			AddRow(2, 2, "bob", 2, "second", now))

	comments, err := repo.GetByBlockID(context.Background(), 2)
	assert.NoError(t, err)
	assert.Len(t, comments, 2)
	assert.Equal(t, "alice", comments[0].Username)
	assert.Equal(t, "bob", comments[1].Username)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCommentRepo_GetByNotebookID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	now := time.Now()
	mock.ExpectQuery(`SELECT.*FROM comments c.*JOIN blocks b.*JOIN users u.*WHERE b.notebook_id`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "username", "block_id", "text", "created_at"}).
			AddRow(1, 1, "alice", 10, "test", now))

	comments, err := repo.GetByNotebookID(context.Background(), 1)
	assert.NoError(t, err)
	assert.Len(t, comments, 1)
	assert.Equal(t, "alice", comments[0].Username)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCommentRepo_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	now := time.Now()
	mock.ExpectQuery(`SELECT.*FROM comments c.*JOIN users u.*WHERE c.id`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "username", "block_id", "text", "created_at"}).
			AddRow(1, 1, "alice", 10, "test", now))

	comment, err := repo.GetByID(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), comment.ID)
	assert.Equal(t, "alice", comment.Username)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCommentRepo_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	mock.ExpectQuery(`SELECT.*FROM comments c.*JOIN users u.*WHERE c.id`).
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetByID(context.Background(), 999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCommentRepo_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	mock.ExpectExec(`DELETE FROM comments WHERE id`).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(context.Background(), 1)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCommentRepo_Delete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewCommentRepository(db)

	mock.ExpectExec(`DELETE FROM comments WHERE id`).
		WithArgs(int64(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Delete(context.Background(), 999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}
