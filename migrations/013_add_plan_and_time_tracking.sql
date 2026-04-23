ALTER TABLE users ADD COLUMN IF NOT EXISTS plan TEXT NOT NULL DEFAULT 'free';
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_time_seconds BIGINT NOT NULL DEFAULT 0;

UPDATE users SET plan = 'admin' WHERE is_admin = TRUE;
UPDATE users SET last_active_at = updated_at WHERE last_active_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_plan ON users(plan);
