package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type SessionRepo struct {
	db redisv9.Cmdable
}

func NewSessionRepository(db redisv9.Cmdable) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) Create(ctx context.Context, session *domain.Session) error {
	start := time.Now()
	if session == nil {
		logger.Error(ctx, "repo.redis.sessions.Create", "error", "session is nil", "duration", time.Since(start))
		return fmt.Errorf("session is nil")
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		err := fmt.Errorf("%w: session already expired", domain.ErrInvalidInput)
		logger.Error(ctx, "repo.redis.sessions.Create", "error", err, "duration", time.Since(start), "session_id", session.ID)
		return err
	}

	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}

	payload, err := json.Marshal(session)
	if err != nil {
		logger.Error(ctx, "repo.redis.sessions.Create", "error", err, "duration", time.Since(start), "session_id", session.ID)
		return fmt.Errorf("marshal session: %w", err)
	}

	created, err := r.db.SetNX(ctx, sessionKey(session.ID), payload, ttl).Result()
	if err != nil {
		logger.Error(ctx, "repo.redis.sessions.Create", "error", err, "duration", time.Since(start), "session_id", session.ID)
		return err
	}
	if !created {
		logger.Error(ctx, "repo.redis.sessions.Create", "error", domain.ErrConflict, "duration", time.Since(start), "session_id", session.ID)
		return domain.ErrConflict
	}

	logger.Info(ctx, "repo.redis.sessions.Create", "duration", time.Since(start), "session_id", session.ID, "user_id", session.UserID)
	return nil
}

func (r *SessionRepo) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	start := time.Now()
	value, err := r.db.Get(ctx, sessionKey(id)).Result()
	if err != nil {
		if errors.Is(err, redisv9.Nil) {
			logger.Error(ctx, "repo.redis.sessions.GetByID", "error", domain.ErrSessionExpired, "duration", time.Since(start), "session_id", id)
			return nil, domain.ErrSessionExpired
		}
		logger.Error(ctx, "repo.redis.sessions.GetByID", "error", err, "duration", time.Since(start), "session_id", id)
		return nil, err
	}

	s := &domain.Session{}
	if err := json.Unmarshal([]byte(value), s); err != nil {
		logger.Error(ctx, "repo.redis.sessions.GetByID", "error", err, "duration", time.Since(start), "session_id", id)
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	logger.Info(ctx, "repo.redis.sessions.GetByID", "duration", time.Since(start), "session_id", id)
	return s, nil
}

func (r *SessionRepo) DeleteByID(ctx context.Context, id string) error {
	start := time.Now()
	err := r.db.Del(ctx, sessionKey(id)).Err()
	if err != nil {
		logger.Error(ctx, "repo.redis.sessions.DeleteByID", "error", err, "duration", time.Since(start), "session_id", id)
		return err
	}
	logger.Info(ctx, "repo.redis.sessions.DeleteByID", "duration", time.Since(start), "session_id", id)
	return nil
}

func sessionKey(id string) string {
	return "session:" + id
}
