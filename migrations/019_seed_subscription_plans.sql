INSERT INTO subscription_plans (name, price, execution_quota, duration_days)
VALUES
    ('pro', 99900,  100000, 30),
    ('max', 199900, 999999, 30)
ON CONFLICT (name) DO UPDATE
SET price           = EXCLUDED.price,
    execution_quota = EXCLUDED.execution_quota,
    duration_days   = EXCLUDED.duration_days;
