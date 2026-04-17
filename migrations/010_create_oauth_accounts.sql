CREATE TABLE IF NOT EXISTS oauth_accounts (
    id          BIGSERIAL   PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider    TEXT        NOT NULL CHECK (char_length(provider) <= 50),
    provider_id TEXT        NOT NULL CHECK (char_length(provider_id) <= 255),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_oauth_provider_id UNIQUE(provider, provider_id)
);

CREATE INDEX idx_oauth_accounts_user_id ON oauth_accounts(user_id);
