CREATE TABLE IF NOT EXISTS subscription_plans (
    id              BIGSERIAL   PRIMARY KEY,
    name            VARCHAR(50) NOT NULL,
    price           INT         NOT NULL,
    execution_quota INT         NOT NULL,
    duration_days   INT         NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT subscription_plans_name_unique UNIQUE (name),
    CONSTRAINT subscription_plans_name_not_empty CHECK (name <> ''),
    CONSTRAINT subscription_plans_price_non_negative CHECK (price >= 0),
    CONSTRAINT subscription_plans_quota_positive CHECK (execution_quota > 0),
    CONSTRAINT subscription_plans_duration_positive CHECK (duration_days > 0)
);

CREATE TABLE IF NOT EXISTS user_subscriptions (
    id                  BIGSERIAL   PRIMARY KEY,
    user_id             BIGINT      NOT NULL,
    plan_id             BIGINT      NOT NULL,
    execution_remaining INT         NOT NULL,
    started_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_subscriptions_user_id_fk
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT user_subscriptions_plan_id_fk
        FOREIGN KEY (plan_id) REFERENCES subscription_plans(id) ON DELETE RESTRICT,
    CONSTRAINT user_subscriptions_remaining_non_negative
        CHECK (execution_remaining >= 0),
    CONSTRAINT user_subscriptions_expires_gt_started
        CHECK (expires_at > started_at)
);

CREATE INDEX IF NOT EXISTS idx_user_subscriptions_user_id ON user_subscriptions(user_id);
