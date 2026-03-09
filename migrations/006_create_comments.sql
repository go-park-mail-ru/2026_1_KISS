CREATE TABLE IF NOT EXISTS comments (
    id         BIGSERIAL   PRIMARY KEY,
    user_id    BIGINT      NOT NULL,
    block_id   BIGINT      NOT NULL,
    text       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT comments_user_id_fk
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT comments_block_id_fk
        FOREIGN KEY (block_id) REFERENCES blocks(id) ON DELETE CASCADE,
    CONSTRAINT comments_text_not_empty CHECK (text <> '')
);

CREATE INDEX IF NOT EXISTS idx_comments_block_id ON comments(block_id);
CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
