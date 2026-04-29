package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	"github.com/lib/pq"
)

const userColumns = `id, username, email, password_hash, avatar_url, status, description, is_admin, plan, last_active_at, total_time_seconds, created_at, updated_at`

type UserRepo struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) (int64, error) {
	start := time.Now()
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		user.Username, user.Email, user.PasswordHash,
	).Scan(&id, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			logger.Error(ctx, "repo.users.Create", "error", domain.ErrConflict, "duration", time.Since(start), "email", user.Email)
			return 0, domain.ErrConflict
		}
		logger.Error(ctx, "repo.users.Create", "error", err, "duration", time.Since(start), "email", user.Email)
		return 0, err
	}
	logger.Info(ctx, "repo.users.Create", "duration", time.Since(start), "user_id", id)
	return id, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	start := time.Now()
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id,
	).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.AvatarURL, &u.Status, &u.Description, &u.IsAdmin,
		&u.Plan, &u.LastActiveAt, &u.TotalTimeSeconds,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(ctx, "repo.users.GetByID", "error", domain.ErrNotFound, "duration", time.Since(start), "user_id", id)
			return nil, domain.ErrNotFound
		}
		logger.Error(ctx, "repo.users.GetByID", "error", err, "duration", time.Since(start), "user_id", id)
		return nil, err
	}
	logger.Info(ctx, "repo.users.GetByID", "duration", time.Since(start), "user_id", id)
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	start := time.Now()
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1`, email,
	).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.AvatarURL, &u.Status, &u.Description, &u.IsAdmin,
		&u.Plan, &u.LastActiveAt, &u.TotalTimeSeconds,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(ctx, "repo.users.GetByEmail", "error", domain.ErrNotFound, "duration", time.Since(start), "email", email)
			return nil, domain.ErrNotFound
		}
		logger.Error(ctx, "repo.users.GetByEmail", "error", err, "duration", time.Since(start), "email", email)
		return nil, err
	}
	logger.Info(ctx, "repo.users.GetByEmail", "duration", time.Since(start), "user_id", u.ID)
	return u, nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	start := time.Now()
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE username = $1`, username,
	).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.AvatarURL, &u.Status, &u.Description, &u.IsAdmin,
		&u.Plan, &u.LastActiveAt, &u.TotalTimeSeconds,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Error(ctx, "repo.users.GetByUsername", "error", domain.ErrNotFound, "duration", time.Since(start), "username", username)
			return nil, domain.ErrNotFound
		}
		logger.Error(ctx, "repo.users.GetByUsername", "error", err, "duration", time.Since(start), "username", username)
		return nil, err
	}
	logger.Info(ctx, "repo.users.GetByUsername", "duration", time.Since(start), "user_id", u.ID)
	return u, nil
}

func (r *UserRepo) UpdateAvatarURL(ctx context.Context, userID int64, avatarURL string) error {
	start := time.Now()
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET avatar_url = $1 WHERE id = $2`,
		avatarURL, userID,
	)
	if err != nil {
		logger.Error(ctx, "repo.users.UpdateAvatarURL", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		logger.Error(ctx, "repo.users.UpdateAvatarURL", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	if n == 0 {
		logger.Error(ctx, "repo.users.UpdateAvatarURL", "error", domain.ErrNotFound, "duration", time.Since(start), "user_id", userID)
		return domain.ErrNotFound
	}
	logger.Info(ctx, "repo.users.UpdateAvatarURL", "duration", time.Since(start), "user_id", userID)
	return nil
}

func (r *UserRepo) UpdateProfile(ctx context.Context, user *domain.User) error {
	start := time.Now()
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET username = $1, status = $2, description = $3 WHERE id = $4`,
		user.Username, user.Status, user.Description, user.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			logger.Error(ctx, "repo.users.UpdateProfile", "error", domain.ErrConflict, "duration", time.Since(start), "user_id", user.ID)
			return domain.ErrConflict
		}
		logger.Error(ctx, "repo.users.UpdateProfile", "error", err, "duration", time.Since(start), "user_id", user.ID)
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		logger.Error(ctx, "repo.users.UpdateProfile", "error", err, "duration", time.Since(start), "user_id", user.ID)
		return err
	}
	if n == 0 {
		logger.Error(ctx, "repo.users.UpdateProfile", "error", domain.ErrNotFound, "duration", time.Since(start), "user_id", user.ID)
		return domain.ErrNotFound
	}
	logger.Info(ctx, "repo.users.UpdateProfile", "duration", time.Since(start), "user_id", user.ID)
	return nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	start := time.Now()
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`,
		passwordHash, userID,
	)
	if err != nil {
		logger.Error(ctx, "repo.users.UpdatePassword", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		logger.Error(ctx, "repo.users.UpdatePassword", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	if n == 0 {
		logger.Error(ctx, "repo.users.UpdatePassword", "error", domain.ErrNotFound, "duration", time.Since(start), "user_id", userID)
		return domain.ErrNotFound
	}
	logger.Info(ctx, "repo.users.UpdatePassword", "duration", time.Since(start), "user_id", userID)
	return nil
}

func (r *UserRepo) UpdateEmail(ctx context.Context, userID int64, email string) error {
	start := time.Now()
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET email = $1 WHERE id = $2`,
		email, userID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			logger.Error(ctx, "repo.users.UpdateEmail", "error", domain.ErrConflict, "duration", time.Since(start), "user_id", userID)
			return domain.ErrConflict
		}
		logger.Error(ctx, "repo.users.UpdateEmail", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		logger.Error(ctx, "repo.users.UpdateEmail", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	if n == 0 {
		logger.Error(ctx, "repo.users.UpdateEmail", "error", domain.ErrNotFound, "duration", time.Since(start), "user_id", userID)
		return domain.ErrNotFound
	}
	logger.Info(ctx, "repo.users.UpdateEmail", "duration", time.Since(start), "user_id", userID)
	return nil
}

func (r *UserRepo) ListAll(ctx context.Context, limit, offset int, search string) ([]domain.User, int, error) {
	start := time.Now()
	var total int
	if search != "" {
		err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM users WHERE username ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%'`,
			search,
		).Scan(&total)
		if err != nil {
			logger.Error(ctx, "repo.users.ListAll.count", "error", err, "duration", time.Since(start))
			return nil, 0, err
		}
	} else {
		err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
		if err != nil {
			logger.Error(ctx, "repo.users.ListAll.count", "error", err, "duration", time.Since(start))
			return nil, 0, err
		}
	}

	var rows *sql.Rows
	var err error
	if search != "" {
		rows, err = r.db.QueryContext(ctx,
			`SELECT `+userColumns+`
			 FROM users WHERE username ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%'
			 ORDER BY id DESC LIMIT $2 OFFSET $3`,
			search, limit, offset,
		)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT `+userColumns+`
			 FROM users ORDER BY id DESC LIMIT $1 OFFSET $2`,
			limit, offset,
		)
	}
	if err != nil {
		logger.Error(ctx, "repo.users.ListAll", "error", err, "duration", time.Since(start))
		return nil, 0, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.PasswordHash,
			&u.AvatarURL, &u.Status, &u.Description, &u.IsAdmin,
			&u.Plan, &u.LastActiveAt, &u.TotalTimeSeconds,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			logger.Error(ctx, "repo.users.ListAll.scan", "error", err, "duration", time.Since(start))
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		logger.Error(ctx, "repo.users.ListAll.rows", "error", err, "duration", time.Since(start))
		return nil, 0, err
	}
	logger.Info(ctx, "repo.users.ListAll", "duration", time.Since(start), "count", len(users), "total", total)
	return users, total, nil
}

func (r *UserRepo) SetBanned(ctx context.Context, userID int64, banned bool) error {
	start := time.Now()
	status := ""
	if banned {
		status = "banned"
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET status = $1 WHERE id = $2`,
		status, userID,
	)
	if err != nil {
		logger.Error(ctx, "repo.users.SetBanned", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		logger.Error(ctx, "repo.users.SetBanned", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	if n == 0 {
		logger.Error(ctx, "repo.users.SetBanned", "error", domain.ErrNotFound, "duration", time.Since(start), "user_id", userID)
		return domain.ErrNotFound
	}
	logger.Info(ctx, "repo.users.SetBanned", "duration", time.Since(start), "user_id", userID, "banned", banned)
	return nil
}

func (r *UserRepo) CountAll(ctx context.Context) (int64, error) {
	start := time.Now()
	var count int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		logger.Error(ctx, "repo.users.CountAll", "error", err, "duration", time.Since(start))
		return 0, err
	}
	logger.Info(ctx, "repo.users.CountAll", "duration", time.Since(start), "count", count)
	return count, nil
}

func (r *UserRepo) UpdatePlan(ctx context.Context, userID int64, plan string) error {
	start := time.Now()
	isAdmin := plan == domain.PlanAdmin
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET plan = $1, is_admin = $2 WHERE id = $3`,
		plan, isAdmin, userID,
	)
	if err != nil {
		logger.Error(ctx, "repo.users.UpdatePlan", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		logger.Error(ctx, "repo.users.UpdatePlan", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	if n == 0 {
		logger.Error(ctx, "repo.users.UpdatePlan", "error", domain.ErrNotFound, "duration", time.Since(start), "user_id", userID)
		return domain.ErrNotFound
	}
	logger.Info(ctx, "repo.users.UpdatePlan", "duration", time.Since(start), "user_id", userID, "plan", plan)
	return nil
}

func (r *UserRepo) UpdateLastActive(ctx context.Context, userID int64, t time.Time) error {
	start := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET last_active_at = $1 WHERE id = $2`,
		t, userID,
	)
	if err != nil {
		logger.Error(ctx, "repo.users.UpdateLastActive", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	logger.Info(ctx, "repo.users.UpdateLastActive", "duration", time.Since(start), "user_id", userID)
	return nil
}

func (r *UserRepo) IncrementTotalTime(ctx context.Context, userID int64, seconds int64) error {
	start := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET total_time_seconds = total_time_seconds + $1 WHERE id = $2`,
		seconds, userID,
	)
	if err != nil {
		logger.Error(ctx, "repo.users.IncrementTotalTime", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	logger.Info(ctx, "repo.users.IncrementTotalTime", "duration", time.Since(start), "user_id", userID, "seconds", seconds)
	return nil
}

func (r *UserRepo) AdminUpdateUser(ctx context.Context, userID int64, username, email string) error {
	start := time.Now()
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET username = $1, email = $2 WHERE id = $3`,
		username, email, userID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			logger.Error(ctx, "repo.users.AdminUpdateUser", "error", domain.ErrConflict, "duration", time.Since(start), "user_id", userID)
			return domain.ErrConflict
		}
		logger.Error(ctx, "repo.users.AdminUpdateUser", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		logger.Error(ctx, "repo.users.AdminUpdateUser", "error", err, "duration", time.Since(start), "user_id", userID)
		return err
	}
	if n == 0 {
		logger.Error(ctx, "repo.users.AdminUpdateUser", "error", domain.ErrNotFound, "duration", time.Since(start), "user_id", userID)
		return domain.ErrNotFound
	}
	logger.Info(ctx, "repo.users.AdminUpdateUser", "duration", time.Since(start), "user_id", userID)
	return nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

func (r *UserRepo) SetVerified(ctx context.Context, userID int64, isVerified bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET is_verified = $1 WHERE id = $2`,
		isVerified, userID,
	)
	return err
}
