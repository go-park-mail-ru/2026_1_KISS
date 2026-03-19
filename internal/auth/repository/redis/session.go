package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type SessionRepo struct {
	db redisv9.Cmdable
}

func NewSessionRepository(db redisv9.Cmdable) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) Create(ctx context.Context, session *domain.Session) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("%w: session already expired", domain.ErrInvalidInput)
	}

	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}

	payload, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	created, err := r.db.SetNX(ctx, sessionKey(session.ID), payload, ttl).Result()
	if err != nil {
		return err
	}
	if !created {
		return domain.ErrConflict
	}

	return nil
}

func (r *SessionRepo) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	value, err := r.db.Get(ctx, sessionKey(id)).Result()
	if err != nil {
		if errors.Is(err, redisv9.Nil) {
			// В Redis TTL удаляет ключ физически, поэтому miss трактуем как истекшую сессию.
			return nil, domain.ErrSessionExpired
		}
		return nil, err
	}

	s := &domain.Session{}
	if err := json.Unmarshal([]byte(value), s); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	return s, nil
}

func (r *SessionRepo) DeleteByID(ctx context.Context, id string) error {
	return r.db.Del(ctx, sessionKey(id)).Err()
}

func sessionKey(id string) string {
	return "session:" + id
}
