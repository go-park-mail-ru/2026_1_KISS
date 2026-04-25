CREATE TYPE category_type AS ENUM ('bug', 'idea', 'problem', 'feedback');

CREATE TABLE IF NOT EXISTS issue (
                                     id BIGSERIAL PRIMARY KEY,
                                     category category_type NOT NULL,
                                     status TEXT NOT NULL,
                                     content TEXT NOT NULL,
                                     created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                                     updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                                     user_id    BIGINT      NOT NULL,
                                     CONSTRAINT user_events_user_id_fk
                                         FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_issue_user_id ON issue(user_id);
CREATE INDEX idx_issue_status ON issue(status);
CREATE INDEX idx_issue_category ON issue(category);

CREATE TRIGGER trg_issue_set_updated_at
    BEFORE UPDATE ON issue
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


--
-- BEGIN;
--
-- DROP TRIGGER IF EXISTS trg_issue_set_updated_at ON issue;
-- DROP INDEX IF EXISTS idx_issue_user_id;
-- DROP INDEX IF EXISTS idx_issue_status;
-- DROP INDEX IF EXISTS idx_issue_category;
-- DROP TABLE IF EXISTS issue CASCADE;
-- DROP TYPE IF EXISTS category_type CASCADE;
--
-- COMMIT;