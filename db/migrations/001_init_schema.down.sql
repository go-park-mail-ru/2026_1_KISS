-- 001_init_schema.down.sql
-- Полный откат схемы базы данных.

DROP TABLE IF EXISTS user_subscription;
DROP TABLE IF EXISTS subscription_plan;
DROP TABLE IF EXISTS comment;
DROP TABLE IF EXISTS cell_output;
DROP TABLE IF EXISTS ipynb_cell;
DROP TABLE IF EXISTS file_permission;
DROP TABLE IF EXISTS ipynb_file;
DROP TABLE IF EXISTS user_account;

DROP FUNCTION IF EXISTS set_updated_at() CASCADE;

DROP TYPE IF EXISTS output_type;
DROP TYPE IF EXISTS cell_type;
DROP TYPE IF EXISTS permission_level;
DROP TYPE IF EXISTS programming_language;
