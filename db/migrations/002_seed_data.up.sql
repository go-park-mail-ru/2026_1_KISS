-- Тестовые учётные записи.
-- Хеш пароля 'password123', bcrypt cost=10 (Default).

INSERT INTO user_account (name, status, description, email, password_hash)
VALUES
    (
        'Alice Admin',
        'active',
        'Администратор',
        'alice@example.com',
        '$2a$10$DjW3ntiNgIU0qY4VXPYz.OaqHA.DweNUQL/2QntXx4R03RND/HDC2'
    ),
    (
        'Bob User',
        'active',
        'Обычный пользователь',
        'bob@example.com',
        '$2a$10$DjW3ntiNgIU0qY4VXPYz.OaqHA.DweNUQL/2QntXx4R03RND/HDC2'
    ),
    (
        'Charlie Coder',
        'active',
        'Data Scientist',
        'charlie@example.com',
        '$2a$10$DjW3ntiNgIU0qY4VXPYz.OaqHA.DweNUQL/2QntXx4R03RND/HDC2'
    );

INSERT INTO subscription_plan (name, price, execution_quota, duration_day)
VALUES
    ('Free',    0,      10,   30),
    ('Premium', 99900,  500,  30),
    ('Annual',  899900, 6000, 365);

INSERT INTO ipynb_file (owner_id, title, nbformat, nbformat_minor, programming_language)
VALUES
    (1, 'Hello World Notebook', 4, 5, 'Python');

INSERT INTO file_permission (file_id, user_id, permission_level)
VALUES
    (1, 2, 'readonly');

INSERT INTO ipynb_cell (file_id, order_index, cell_type, source)
VALUES
    (1, 1, 'markdown', '# Hello World'),
    (1, 2, 'code',     'print("Hello, KISS!")'),
    (1, 3, 'code',     'x = 42' || chr(10) || 'print(f"Answer: {x}")');

INSERT INTO comment (user_id, cell_id, text)
VALUES
    (2, 1, 'Отличное начало!');

INSERT INTO user_subscription (user_id, plan_id, execution_remaining, started_at, expires_at)
VALUES
    (1, 1, 10, now(), now() + INTERVAL '30 days');
