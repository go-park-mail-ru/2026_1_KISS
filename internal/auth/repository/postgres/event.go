package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type EventRepo struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepo {
	return &EventRepo{db: db}
}

func (r *EventRepo) Create(ctx context.Context, event *domain.UserEvent) error {
	start := time.Now()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_events (user_id, event_type, metadata) VALUES ($1, $2, $3)`,
		event.UserID, event.EventType, event.Metadata,
	)
	if err != nil {
		logger.Error(ctx, "repo.events.Create", "error", err, "duration", time.Since(start), "user_id", event.UserID)
		return err
	}
	logger.Info(ctx, "repo.events.Create", "duration", time.Since(start), "user_id", event.UserID, "type", event.EventType)
	return nil
}

func (r *EventRepo) CountActiveUsers(ctx context.Context, since time.Time) (int64, error) {
	start := time.Now()
	var count int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM user_events WHERE created_at >= $1`,
		since,
	).Scan(&count)
	if err != nil {
		logger.Error(ctx, "repo.events.CountActiveUsers", "error", err, "duration", time.Since(start))
		return 0, err
	}
	logger.Info(ctx, "repo.events.CountActiveUsers", "duration", time.Since(start), "count", count)
	return count, nil
}
