ALTER TABLE files
    ADD COLUMN IF NOT EXISTS is_public        BOOLEAN     NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS share_token      UUID        NULL,
    ADD COLUMN IF NOT EXISTS share_expires_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS downloads_count  BIGINT      NOT NULL DEFAULT 0;

CREATE UNIQUE INDEX IF NOT EXISTS idx_files_share_token
    ON files (share_token)
    WHERE share_token IS NOT NULL;

CREATE TABLE IF NOT EXISTS file_shares (
    file_id          UUID        NOT NULL,
    user_id          BIGINT      NOT NULL,
    permission_level VARCHAR(20) NOT NULL DEFAULT 'view',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT file_shares_pk PRIMARY KEY (file_id, user_id),
    CONSTRAINT file_shares_file_id_fk
        FOREIGN KEY (file_id) REFERENCES files (id) ON DELETE CASCADE,
    CONSTRAINT file_shares_user_id_fk
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT file_shares_level_check
        CHECK (permission_level IN ('view', 'download'))
);

CREATE INDEX IF NOT EXISTS idx_file_shares_user_id ON file_shares (user_id);
