-- 002_seed_data.down.sql
-- Откат: очистка данных.

TRUNCATE TABLE
    user_subscription,
    comment,
    cell_output,
    ipynb_cell,
    file_permission,
    ipynb_file,
    subscription_plan,
    user_account
RESTART IDENTITY;
