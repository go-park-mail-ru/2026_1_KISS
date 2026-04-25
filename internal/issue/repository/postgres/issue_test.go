package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestIssueRepo_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := &issueRepo{db: db}
		now := time.Now()

		mock.ExpectQuery(`SELECT id, category, status, content, created_at, updated_at, user_id FROM issue WHERE id = \$1`).
			WithArgs(int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "category", "status", "content", "created_at", "updated_at", "user_id"}).
				AddRow(int64(7), "bug", "open", "Test issue content", now, now, int64(100)))

		issue, err := repo.GetByID(context.Background(), 7)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		assert.Equal(t, int64(7), issue.ID)
		assert.Equal(t, domain.IssueCategory("bug"), issue.Category)
		assert.Equal(t, domain.IssueStatus("open"), issue.Status)
		assert.Equal(t, "Test issue content", issue.Content)
		assert.Equal(t, int64(100), issue.UserID)
		assert.Equal(t, now, issue.CreatedAt)
		assert.Equal(t, now, issue.UpdatedAt)

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

		repo := &issueRepo{db: db}

		mock.ExpectQuery(`SELECT id, category, status, content, created_at, updated_at, user_id FROM issue WHERE id = \$1`).
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

func TestIssueRepo_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := &issueRepo{db: db}
		now := time.Now()
		issue := &domain.Issue{
			Category: domain.IssueCategory("bug"),
			Status:   domain.IssueStatus("open"),
			Content:  "New issue content",
			UserID:   100,
		}

		mock.ExpectQuery(`INSERT INTO issue \(category, status, content, user_id\) VALUES \(\$1, \$2, \$3, \$4\) RETURNING id, created_at, updated_at`).
			WithArgs(issue.Category, issue.Status, issue.Content, issue.UserID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(int64(10), now, now))

		id, err := repo.Create(context.Background(), issue)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		assert.Equal(t, int64(10), id)
		assert.Equal(t, now, issue.CreatedAt)
		assert.Equal(t, now, issue.UpdatedAt)

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestIssueRepo_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := &issueRepo{db: db}
		now := time.Now()
		issue := &domain.Issue{
			ID:       7,
			Category: domain.IssueCategory("idea"),
			Status:   domain.IssueStatus("in_progress"),
			Content:  "Updated content",
			UserID:   100,
		}

		mock.ExpectQuery(`UPDATE issue SET category = \$1, status = \$2, content = \$3, updated_at = NOW\(\) WHERE id = \$4 AND user_id = \$5 RETURNING updated_at`).
			WithArgs(issue.Category, issue.Status, issue.Content, issue.ID, issue.UserID).
			WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(now))

		err = repo.Update(context.Background(), issue)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		assert.Equal(t, now, issue.UpdatedAt)

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

		repo := &issueRepo{db: db}
		issue := &domain.Issue{
			ID:     999,
			UserID: 100,
		}

		mock.ExpectQuery(`UPDATE issue SET category = \$1, status = \$2, content = \$3, updated_at = NOW\(\) WHERE id = \$4 AND user_id = \$5 RETURNING updated_at`).
			WithArgs(issue.Category, issue.Status, issue.Content, issue.ID, issue.UserID).
			WillReturnError(sql.ErrNoRows)

		err = repo.Update(context.Background(), issue)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestIssueRepo_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := &issueRepo{db: db}

		mock.ExpectExec(`DELETE FROM issue WHERE id = \$1`).
			WithArgs(int64(7)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err = repo.Delete(context.Background(), 7)
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

		repo := &issueRepo{db: db}

		mock.ExpectExec(`DELETE FROM issue WHERE id = \$1`).
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

func TestIssueRepo_GetAll(t *testing.T) {
	t.Run("success with filters", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := &issueRepo{db: db}
		now := time.Now()
		filter := &domain.IssueFilter{
			ID:        7,
			Category:  "bug",
			Status:    "open",
			UserID:    100,
			Content:   "test",
			CreatedAt: now,
			UpdatedAt: now,
		}

		rows := sqlmock.NewRows([]string{"id", "category", "status", "content", "created_at", "updated_at", "user_id"}).
			AddRow(int64(7), "bug", "open", "test content", now, now, int64(100)).
			AddRow(int64(8), "bug", "open", "another test", now, now, int64(100))

		mock.ExpectQuery(`SELECT id, category, status, content, created_at, updated_at, user_id FROM issue WHERE 1=1 AND id = \$1 AND category = \$2 AND status = \$3 AND user_id = \$4 AND content ILIKE '%' \|\| \$5 \|\| '%' AND created_at >= \$6 AND updated_at >= \$7 ORDER BY id DESC LIMIT \$8 OFFSET \$9`).
			WithArgs(filter.ID, filter.Category, filter.Status, filter.UserID, filter.Content, filter.CreatedAt, filter.UpdatedAt, 10, 0).
			WillReturnRows(rows)

		issues, err := repo.GetAll(context.Background(), 10, 0, filter)
		if err != nil {
			t.Fatalf("GetAll() error = %v", err)
		}
		assert.Len(t, issues, 2)
		assert.Equal(t, int64(7), issues[0].ID)
		assert.Equal(t, domain.IssueCategory("bug"), issues[0].Category)
		assert.Equal(t, domain.IssueStatus("open"), issues[0].Status)

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("success without filters", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := &issueRepo{db: db}
		now := time.Now()

		rows := sqlmock.NewRows([]string{"id", "category", "status", "content", "created_at", "updated_at", "user_id"}).
			AddRow(int64(1), "bug", "open", "content1", now, now, int64(100))

		mock.ExpectQuery(`SELECT id, category, status, content, created_at, updated_at, user_id FROM issue WHERE 1=1 ORDER BY id DESC LIMIT \$1 OFFSET \$2`).
			WithArgs(10, 0).
			WillReturnRows(rows)

		issues, err := repo.GetAll(context.Background(), 10, 0, nil)
		if err != nil {
			t.Fatalf("GetAll() error = %v", err)
		}
		assert.Len(t, issues, 1)

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("empty result", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := &issueRepo{db: db}

		mock.ExpectQuery(`SELECT id, category, status, content, created_at, updated_at, user_id FROM issue WHERE 1=1 ORDER BY id DESC LIMIT \$1 OFFSET \$2`).
			WithArgs(10, 0).
			WillReturnRows(sqlmock.NewRows([]string{"id", "category", "status", "content", "created_at", "updated_at", "user_id"}))

		issues, err := repo.GetAll(context.Background(), 10, 0, nil)
		if err != nil {
			t.Fatalf("GetAll() error = %v", err)
		}
		assert.Empty(t, issues)

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}
