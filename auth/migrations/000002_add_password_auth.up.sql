-- Add support for username/password authentication alongside OAuth

-- Add username and password_hash columns
ALTER TABLE users
    ADD COLUMN username VARCHAR(50) UNIQUE,
    ADD COLUMN password_hash VARCHAR(255);

-- Make OAuth fields nullable (for password-only users)
ALTER TABLE users
    ALTER COLUMN oauth_provider DROP NOT NULL,
    ALTER COLUMN oauth_id DROP NOT NULL;

-- Drop the old unique constraint and create a new one that allows nulls
ALTER TABLE users DROP CONSTRAINT users_oauth_provider_oauth_id_key;
ALTER TABLE users ADD CONSTRAINT users_oauth_unique
    UNIQUE NULLS NOT DISTINCT (oauth_provider, oauth_id);
