CREATE TABLE IF NOT EXISTS issue_message (
    id         BIGSERIAL    PRIMARY KEY,
    issue_id   BIGINT       NOT NULL REFERENCES issue(id) ON DELETE CASCADE,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_admin   BOOLEAN      NOT NULL DEFAULT FALSE,
    content    TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_issue_message_issue_id ON issue_message(issue_id);
CREATE INDEX idx_issue_message_user_id ON issue_message(user_id);

-- Rollback:
-- DROP INDEX IF EXISTS idx_issue_message_user_id;
-- DROP INDEX IF EXISTS idx_issue_message_issue_id;
-- DROP TABLE IF EXISTS issue_message CASCADE;
