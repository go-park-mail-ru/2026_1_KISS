CREATE TYPE programming_language AS ENUM (
    'Python',
    'R'
);

CREATE TYPE permission_level AS ENUM (
    'edit',
    'comment',
    'readonly'
);

CREATE TYPE cell_type AS ENUM (
    'code',
    'markdown',
    'raw'
);

CREATE TYPE output_type AS ENUM (
    'stream',
    'execute_result',
    'error',
    'display_data'
);

-- Триггерная функция для автоматического обновления updated_at.

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;


-- Используем bcrypt, чтобы руками не солить пароли! 
-- И тогда поле с солью не нужно!

CREATE TABLE user_account (
    id              integer        GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name            text           NOT NULL,
    status          text           NOT NULL DEFAULT '',
    description     text           NOT NULL DEFAULT '',
    email           text           NOT NULL,
    password_hash   text           NOT NULL,
    avatar_url      text           NOT NULL DEFAULT '',
    created_at      timestamptz    NOT NULL DEFAULT now(),
    updated_at      timestamptz    NOT NULL DEFAULT now(),

    CONSTRAINT user_account_email_unique
        UNIQUE (email),

    CONSTRAINT user_account_email_not_empty
        CHECK (email <> ''),

    CONSTRAINT user_account_name_not_empty
        CHECK (name <> ''),

    CONSTRAINT user_account_password_hash_not_empty
        CHECK (password_hash <> ''),

    CONSTRAINT user_account_updated_gte_created
        CHECK (updated_at >= created_at)
);

CREATE TRIGGER trg_user_account_updated_at
    BEFORE UPDATE ON user_account
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();


-- Notebooks

CREATE TABLE ipynb_file (
    id                      integer                GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_id                integer                NOT NULL,
    title                   text                   NOT NULL DEFAULT 'Untitled',
    nbformat                integer                NOT NULL DEFAULT 4,
    nbformat_minor          integer                NOT NULL DEFAULT 5,
    programming_language    programming_language   NOT NULL DEFAULT 'Python',
    created_at              timestamptz            NOT NULL DEFAULT now(),
    updated_at              timestamptz            NOT NULL DEFAULT now(),

    CONSTRAINT ipynb_file_owner_id_fk
        FOREIGN KEY (owner_id)
        REFERENCES user_account (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    CONSTRAINT ipynb_file_title_not_empty
        CHECK (title <> ''),

    CONSTRAINT ipynb_file_nbformat_positive
        CHECK (nbformat > 0),

    CONSTRAINT ipynb_file_nbformat_minor_non_negative
        CHECK (nbformat_minor >= 0),

    CONSTRAINT ipynb_file_updated_gte_created
        CHECK (updated_at >= created_at)
);

CREATE INDEX idx_ipynb_file_owner_id
    ON ipynb_file (owner_id);

CREATE TRIGGER trg_ipynb_file_updated_at
    BEFORE UPDATE ON ipynb_file
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();


-- Permissions

CREATE TABLE file_permission (
    file_id             integer            NOT NULL,
    user_id             integer            NOT NULL,
    permission_level    permission_level   NOT NULL DEFAULT 'readonly',

    CONSTRAINT file_permission_pk
        PRIMARY KEY (file_id, user_id),

    CONSTRAINT file_permission_file_id_fk
        FOREIGN KEY (file_id)
        REFERENCES ipynb_file (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    CONSTRAINT file_permission_user_id_fk
        FOREIGN KEY (user_id)
        REFERENCES user_account (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

CREATE INDEX idx_file_permission_user_id
    ON file_permission (user_id);


-- Cells

CREATE TABLE ipynb_cell (
    id                  integer        GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    file_id             integer        NOT NULL,
    order_index         integer        NOT NULL,
    cell_type           cell_type      NOT NULL DEFAULT 'code',
    source              text           NOT NULL DEFAULT '',
    execution_count     integer,
    created_at          timestamptz    NOT NULL DEFAULT now(),
    updated_at          timestamptz    NOT NULL DEFAULT now(),

    CONSTRAINT ipynb_cell_file_id_fk
        FOREIGN KEY (file_id)
        REFERENCES ipynb_file (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    CONSTRAINT ipynb_cell_file_order_unique
        UNIQUE (file_id, order_index),

    CONSTRAINT ipynb_cell_order_index_positive
        CHECK (order_index > 0),

    CONSTRAINT ipynb_cell_execution_count_positive
        CHECK (execution_count IS NULL OR execution_count > 0),

    CONSTRAINT ipynb_cell_updated_gte_created
        CHECK (updated_at >= created_at)
);

CREATE TRIGGER trg_ipynb_cell_updated_at
    BEFORE UPDATE ON ipynb_cell
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();


-- Outputs

CREATE TABLE cell_output (
    id              integer        GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cell_id         integer        NOT NULL,
    order_index     integer        NOT NULL,
    output_type     output_type    NOT NULL,
    text_content    text           NOT NULL DEFAULT '',
    created_at      timestamptz    NOT NULL DEFAULT now(),

    CONSTRAINT cell_output_cell_id_fk
        FOREIGN KEY (cell_id)
        REFERENCES ipynb_cell (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    CONSTRAINT cell_output_cell_order_unique
        UNIQUE (cell_id, order_index),

    CONSTRAINT cell_output_order_index_positive
        CHECK (order_index > 0)
);

CREATE INDEX idx_cell_output_cell_id
    ON cell_output (cell_id);


-- Comments

CREATE TABLE comment (
    id              integer        GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id         integer        NOT NULL,
    cell_id         integer        NOT NULL,
    text            text           NOT NULL,
    created_at      timestamptz    NOT NULL DEFAULT now(),

    CONSTRAINT comment_user_id_fk
        FOREIGN KEY (user_id)
        REFERENCES user_account (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    CONSTRAINT comment_cell_id_fk
        FOREIGN KEY (cell_id)
        REFERENCES ipynb_cell (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    CONSTRAINT comment_text_not_empty
        CHECK (text <> '')
);

CREATE INDEX idx_comment_cell_id
    ON comment (cell_id);

CREATE INDEX idx_comment_user_id
    ON comment (user_id);


-- Subscription plans

CREATE TABLE subscription_plan (
    id                  integer        GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name                text           NOT NULL,
    price               integer        NOT NULL,
    execution_quota     integer        NOT NULL,
    duration_day        integer        NOT NULL,
    created_at          timestamptz    NOT NULL DEFAULT now(),

    CONSTRAINT subscription_plan_name_unique
        UNIQUE (name),

    CONSTRAINT subscription_plan_name_not_empty
        CHECK (name <> ''),

    CONSTRAINT subscription_plan_price_non_negative
        CHECK (price >= 0),

    CONSTRAINT subscription_plan_execution_quota_positive
        CHECK (execution_quota > 0),

    CONSTRAINT subscription_plan_duration_day_positive
        CHECK (duration_day > 0)
);


-- User subscriptions

CREATE TABLE user_subscription (
    id                      integer        GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id                 integer        NOT NULL,
    plan_id                 integer        NOT NULL,
    execution_remaining     integer        NOT NULL,
    started_at              timestamptz    NOT NULL DEFAULT now(),
    expires_at              timestamptz    NOT NULL,
    created_at              timestamptz    NOT NULL DEFAULT now(),

    CONSTRAINT user_subscription_user_id_fk
        FOREIGN KEY (user_id)
        REFERENCES user_account (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    CONSTRAINT user_subscription_plan_id_fk
        FOREIGN KEY (plan_id)
        REFERENCES subscription_plan (id)
        ON DELETE RESTRICT
        ON UPDATE CASCADE,

    CONSTRAINT user_subscription_execution_remaining_non_negative
        CHECK (execution_remaining >= 0),

    CONSTRAINT user_subscription_expires_gt_started
        CHECK (expires_at > started_at)
);

CREATE INDEX idx_user_subscription_user_id
    ON user_subscription (user_id);
