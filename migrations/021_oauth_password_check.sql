ALTER TABLE users DROP CONSTRAINT IF EXISTS users_password_hash_not_empty;
ALTER TABLE users
    ADD CONSTRAINT users_password_hash_check
    CHECK (password_hash IS NULL OR password_hash <> '');
