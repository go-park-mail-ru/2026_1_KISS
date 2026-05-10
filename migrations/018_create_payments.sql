CREATE TABLE IF NOT EXISTS payments (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             BIGINT      NOT NULL,
    plan_id             BIGINT      NOT NULL,
    yookassa_payment_id TEXT,
    status              TEXT        NOT NULL DEFAULT 'pending',
    amount_kopeks       BIGINT      NOT NULL,
    currency            VARCHAR(3)  NOT NULL DEFAULT 'RUB',
    confirmation_token  TEXT,
    idempotence_key     UUID        NOT NULL,
    description         TEXT        NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at             TIMESTAMPTZ,
    CONSTRAINT payments_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT payments_plan_fk FOREIGN KEY (plan_id) REFERENCES subscription_plans(id) ON DELETE RESTRICT,
    CONSTRAINT payments_status_check CHECK (status IN ('pending','succeeded','canceled','waiting_for_capture')),
    CONSTRAINT payments_amount_non_negative CHECK (amount_kopeks >= 0),
    CONSTRAINT payments_idempotence_unique UNIQUE (idempotence_key),
    CONSTRAINT payments_yookassa_unique UNIQUE (yookassa_payment_id)
);

CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_yookassa_payment_id ON payments(yookassa_payment_id);
