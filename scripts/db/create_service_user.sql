-- scripts/db/create_service_user.sql
--   psql "$DATABASE_URL" -f scripts/db/create_service_user.sql

--auth-сервис
--    Читает и пишет: users, sessions, oauth_accounts,
--    verification_tokens, user_events, user_subscriptions,
--    subscription_plans
--    Не трогает: notebooks, blocks, block_outputs, issue, files
CREATE USER colab_auth WITH
    PASSWORD 'CHANGE_ME_AUTH_PASSWORD'
    CONNECTION LIMIT 30
    NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT LOGIN;

GRANT USAGE ON SCHEMA public TO colab_auth;

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE
    users,
    sessions,
    oauth_accounts,
    verification_tokens,
    user_events,
    user_subscriptions,
    subscription_plans
TO colab_auth;

GRANT USAGE, SELECT ON SEQUENCE
    users_id_seq,
    user_events_id_seq,
    user_subscriptions_id_seq,
    subscription_plans_id_seq,
    verification_tokens_id_seq
TO colab_auth;

--notebook-сервис
--    Читает и пишет: notebooks, blocks, block_outputs,
--    file_permissions, comments
--    Читает (JOIN): users (только SELECT для проверки owner)
CREATE USER colab_notebook WITH
    PASSWORD 'CHANGE_ME_NOTEBOOK_PASSWORD'
    CONNECTION LIMIT 30
    NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT LOGIN;

GRANT USAGE ON SCHEMA public TO colab_notebook;

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE
    notebooks,
    blocks,
    block_outputs,
    file_permissions,
    comments
TO colab_notebook;

GRANT SELECT ON TABLE users TO colab_notebook;

GRANT USAGE, SELECT ON SEQUENCE
    notebooks_id_seq,
    blocks_id_seq,
    block_outputs_id_seq,
    comments_id_seq
TO colab_notebook;

--storage-сервис
--    Читает и пишет: files
--    Читает: users (проверка owner)
CREATE USER colab_storage WITH
    PASSWORD 'CHANGE_ME_STORAGE_PASSWORD'
    CONNECTION LIMIT 30
    NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT LOGIN;

GRANT USAGE ON SCHEMA public TO colab_storage;

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE files TO colab_storage;
GRANT SELECT ON TABLE users TO colab_storage;


--issue-сервис
--    Читает и пишет: issue, issue_message
--    Читает: users
CREATE USER colab_issue WITH
    PASSWORD 'CHANGE_ME_ISSUE_PASSWORD'
    CONNECTION LIMIT 30
    NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT LOGIN;

GRANT USAGE ON SCHEMA public TO colab_issue;

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE
    issue,
    issue_message
TO colab_issue;

GRANT SELECT ON TABLE users TO colab_issue;

GRANT USAGE, SELECT ON SEQUENCE
    issue_id_seq,
    issue_message_id_seq
TO colab_issue;

--Запрет CREATE на схему для всех сервисных пользователей
--    (DDL — только через мигратор под суперпользователем)
REVOKE CREATE ON SCHEMA public FROM colab_auth;
REVOKE CREATE ON SCHEMA public FROM colab_notebook;
REVOKE CREATE ON SCHEMA public FROM colab_storage;
REVOKE CREATE ON SCHEMA public FROM colab_issue;

-- Проверка выданных прав:
-- SELECT grantee, table_name, privilege_type
-- FROM information_schema.role_table_grants
-- WHERE grantee LIKE 'colab_%'
-- ORDER BY grantee, table_name
