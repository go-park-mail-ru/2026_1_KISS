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

type OAuthStateRepo struct {
	db redisv9.Cmdable
}

func NewOAuthStateRepository(db redisv9.Cmdable) *OAuthStateRepo {
	return &OAuthStateRepo{db: db}
}

func oauthStateKey(state string) string {
	return "oauth:state:" + state
}

func (r *OAuthStateRepo) Save(ctx context.Context, st *domain.OAuthState, ttl time.Duration) error {
	start := time.Now()
	if st == nil {
		return fmt.Errorf("%w: state is nil", domain.ErrInvalidInput)
	}
	if st.State == "" {
		return fmt.Errorf("%w: empty state", domain.ErrInvalidInput)
	}
	if ttl <= 0 {
		return fmt.Errorf("%w: ttl must be > 0", domain.ErrInvalidInput)
	}
	if st.CreatedAt.IsZero() {
		st.CreatedAt = time.Now().UTC()
	}

	payload, err := json.Marshal(st)
	if err != nil {
		logger.Error(ctx, "repo.redis.oauth_state.Save", "error", err, "duration", time.Since(start))
		return fmt.Errorf("marshal oauth state: %w", err)
	}

	created, err := r.db.SetNX(ctx, oauthStateKey(st.State), payload, ttl).Result()
	if err != nil {
		logger.Error(ctx, "repo.redis.oauth_state.Save", "error", err, "duration", time.Since(start))
		return err
	}
	if !created {
		logger.Error(ctx, "repo.redis.oauth_state.Save", "error", domain.ErrConflict, "duration", time.Since(start))
		return domain.ErrConflict
	}

	logger.Info(ctx, "repo.redis.oauth_state.Save", "duration", time.Since(start), "provider", st.Provider)
	return nil
}

func (r *OAuthStateRepo) Consume(ctx context.Context, state string) (*domain.OAuthState, error) {
	start := time.Now()
	if state == "" {
		return nil, fmt.Errorf("%w: empty state", domain.ErrInvalidInput)
	}

	value, err := r.db.GetDel(ctx, oauthStateKey(state)).Result()
	if err != nil {
		if errors.Is(err, redisv9.Nil) {
			logger.Error(ctx, "repo.redis.oauth_state.Consume", "error", domain.ErrNotFound, "duration", time.Since(start))
			return nil, domain.ErrNotFound
		}
		logger.Error(ctx, "repo.redis.oauth_state.Consume", "error", err, "duration", time.Since(start))
		return nil, err
	}

	st := &domain.OAuthState{}
	if err := json.Unmarshal([]byte(value), st); err != nil {
		logger.Error(ctx, "repo.redis.oauth_state.Consume", "error", err, "duration", time.Since(start))
		return nil, fmt.Errorf("unmarshal oauth state: %w", err)
	}

	logger.Info(ctx, "repo.redis.oauth_state.Consume", "duration", time.Since(start), "provider", st.Provider)
	return st, nil
}
