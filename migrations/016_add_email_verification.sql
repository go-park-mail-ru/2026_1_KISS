ALTER TABLE users ADD COLUMN is_verified BOOLEAN DEFAULT FALSE;

CREATE TABLE verification_tokens (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL
);
