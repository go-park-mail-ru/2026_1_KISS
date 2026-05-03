CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL    PRIMARY KEY,
    username      VARCHAR(50)  NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    avatar_url    TEXT         NOT NULL DEFAULT '',
    status        TEXT         NOT NULL DEFAULT '',
    description   TEXT         NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT users_username_unique UNIQUE (username),
    CONSTRAINT users_email_unique UNIQUE (email),
    CONSTRAINT users_username_not_empty CHECK (username <> ''),
    CONSTRAINT users_email_not_empty CHECK (email <> ''),
    CONSTRAINT users_password_hash_not_empty CHECK (password_hash <> '')
);
