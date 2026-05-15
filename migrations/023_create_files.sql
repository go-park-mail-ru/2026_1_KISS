CREATE TABLE IF NOT EXISTS files (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    BIGINT NOT NULL,
    notebook_id BIGINT,
    category    TEXT NOT NULL DEFAULT 'files',
    filename    TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    url         TEXT NOT NULL,
    mime_type   VARCHAR(255) NOT NULL DEFAULT '',
    size        BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT files_owner_fk FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT files_notebook_fk FOREIGN KEY (notebook_id) REFERENCES notebooks(id) ON DELETE SET NULL,
    CONSTRAINT files_category_check CHECK (category IN ('avatars','feedback','datasets','files'))
);

CREATE INDEX IF NOT EXISTS idx_files_owner_id ON files(owner_id);
CREATE INDEX IF NOT EXISTS idx_files_notebook_id ON files(notebook_id);
CREATE INDEX IF NOT EXISTS idx_files_category ON files(category);
