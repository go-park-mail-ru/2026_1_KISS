CREATE TABLE IF NOT EXISTS notebooks (
    id         BIGSERIAL    PRIMARY KEY,
    owner_id   BIGINT       NOT NULL,
    title      VARCHAR(255) NOT NULL DEFAULT 'Untitled',
    is_public  BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT notebooks_owner_id_fk
        FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT notebooks_title_not_empty CHECK (title <> '')
);

CREATE INDEX IF NOT EXISTS idx_notebooks_owner_id ON notebooks(owner_id);

CREATE TABLE IF NOT EXISTS blocks (
    id              BIGSERIAL   PRIMARY KEY,
    notebook_id     BIGINT      NOT NULL,
    type            VARCHAR(20) NOT NULL DEFAULT 'code',
    language        VARCHAR(20) NOT NULL DEFAULT 'python',
    content         TEXT        NOT NULL DEFAULT '',
    position        INT         NOT NULL DEFAULT 0,
    execution_count INT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT blocks_notebook_id_fk
        FOREIGN KEY (notebook_id) REFERENCES notebooks(id) ON DELETE CASCADE,
    CONSTRAINT blocks_notebook_position_unique UNIQUE (notebook_id, position),
    CONSTRAINT blocks_position_non_negative CHECK (position >= 0),
    CONSTRAINT blocks_execution_count_positive
        CHECK (execution_count IS NULL OR execution_count > 0)
);

CREATE INDEX IF NOT EXISTS idx_blocks_notebook_id ON blocks(notebook_id);
