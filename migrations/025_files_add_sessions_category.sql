ALTER TABLE files DROP CONSTRAINT IF EXISTS files_category_check;
ALTER TABLE files ADD CONSTRAINT files_category_check
    CHECK (category IN ('avatars', 'feedback', 'datasets', 'files', 'sessions'));
