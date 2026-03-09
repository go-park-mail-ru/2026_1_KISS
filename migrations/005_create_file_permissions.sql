CREATE TABLE IF NOT EXISTS file_permissions (
    notebook_id      BIGINT      NOT NULL,
    user_id          BIGINT      NOT NULL,
    permission_level VARCHAR(20) NOT NULL DEFAULT 'readonly',
    CONSTRAINT file_permissions_pk PRIMARY KEY (notebook_id, user_id),
    CONSTRAINT file_permissions_notebook_id_fk
        FOREIGN KEY (notebook_id) REFERENCES notebooks(id) ON DELETE CASCADE,
    CONSTRAINT file_permissions_user_id_fk
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_file_permissions_user_id ON file_permissions(user_id);
