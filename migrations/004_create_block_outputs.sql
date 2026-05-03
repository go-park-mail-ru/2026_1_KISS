CREATE TABLE IF NOT EXISTS block_outputs (
    id          BIGSERIAL   PRIMARY KEY,
    block_id    BIGINT      NOT NULL,
    position    INT         NOT NULL,
    output_type VARCHAR(20) NOT NULL,
    content     TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT block_outputs_block_id_fk
        FOREIGN KEY (block_id) REFERENCES blocks(id) ON DELETE CASCADE,
    CONSTRAINT block_outputs_block_position_unique UNIQUE (block_id, position),
    CONSTRAINT block_outputs_position_non_negative CHECK (position >= 0)
);

CREATE INDEX IF NOT EXISTS idx_block_outputs_block_id ON block_outputs(block_id);
