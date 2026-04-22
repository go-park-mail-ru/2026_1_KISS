package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) Create(ctx context.Context, session *domain.Session) error {
	start := time.Now()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		session.ID, session.UserID, session.ExpiresAt,
	)
	if err != nil {
		logger.Error(ctx, "repo.sessions.Create", "error", err, "duration", time.Since(start), "session_id", session.ID, "user_id", session.UserID)
		return err
	}
	logger.Info(ctx, "repo.sessions.Create", "duration", time.Since(start), "session_id", session.ID, "user_id", session.UserID)
	return nil
}

func (r *SessionRepo) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	start := time.Now()
	s := &domain.Session{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(ctx, "repo.sessions.GetByID", "error", domain.ErrNotFound, "duration", time.Since(start), "session_id", id)
			return nil, domain.ErrNotFound
		}
		logger.Error(ctx, "repo.sessions.GetByID", "error", err, "duration", time.Since(start), "session_id", id)
		return nil, err
	}
	logger.Info(ctx, "repo.sessions.GetByID", "duration", time.Since(start), "session_id", id)
	return s, nil
}

func (r *SessionRepo) DeleteByID(ctx context.Context, id string) error {
	start := time.Now()
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	if err != nil {
		logger.Error(ctx, "repo.sessions.DeleteByID", "error", err, "duration", time.Since(start), "session_id", id)
		return err
	}
	logger.Info(ctx, "repo.sessions.DeleteByID", "duration", time.Since(start), "session_id", id)
	return nil
}
