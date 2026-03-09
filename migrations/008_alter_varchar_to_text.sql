ALTER TABLE users
    ALTER COLUMN username TYPE TEXT;

ALTER TABLE users
    ADD CONSTRAINT users_username_max_length CHECK (length(username) <= 50);


ALTER TABLE users
    ALTER COLUMN email TYPE TEXT;

ALTER TABLE users
    ADD CONSTRAINT users_email_max_length CHECK (length(email) <= 255);


ALTER TABLE users
    ALTER COLUMN password_hash TYPE TEXT;

ALTER TABLE users
    ADD CONSTRAINT users_password_hash_max_length CHECK (length(password_hash) <= 255);


ALTER TABLE notebooks
    ALTER COLUMN title TYPE TEXT;

ALTER TABLE notebooks
    ADD CONSTRAINT notebooks_title_max_length CHECK (length(title) <= 255);


ALTER TABLE blocks
    ALTER COLUMN type TYPE TEXT;

ALTER TABLE blocks
    ADD CONSTRAINT blocks_type_not_empty CHECK (type <> '');

ALTER TABLE blocks
    ADD CONSTRAINT blocks_type_max_length CHECK (length(type) <= 20);


ALTER TABLE blocks
    ALTER COLUMN language TYPE TEXT;

ALTER TABLE blocks
    ADD CONSTRAINT blocks_language_not_empty CHECK (language <> '');

ALTER TABLE blocks
    ADD CONSTRAINT blocks_language_max_length CHECK (length(language) <= 20);


ALTER TABLE block_outputs
    ALTER COLUMN output_type TYPE TEXT;

ALTER TABLE block_outputs
    ADD CONSTRAINT block_outputs_output_type_not_empty CHECK (output_type <> '');

ALTER TABLE block_outputs
    ADD CONSTRAINT block_outputs_output_type_max_length CHECK (length(output_type) <= 20);



ALTER TABLE file_permissions
    ALTER COLUMN permission_level TYPE TEXT;

ALTER TABLE file_permissions
    ADD CONSTRAINT file_permissions_permission_level_not_empty CHECK (permission_level <> '');

ALTER TABLE file_permissions
    ADD CONSTRAINT file_permissions_permission_level_max_length CHECK (length(permission_level) <= 20);


ALTER TABLE subscription_plans
    ALTER COLUMN name TYPE TEXT;

ALTER TABLE subscription_plans
    ADD CONSTRAINT subscription_plans_name_max_length CHECK (length(name) <= 50);
