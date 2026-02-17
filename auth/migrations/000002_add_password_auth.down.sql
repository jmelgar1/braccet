-- Revert password auth changes

-- Drop the new constraint and restore the old one
ALTER TABLE users DROP CONSTRAINT users_oauth_unique;
ALTER TABLE users ADD CONSTRAINT users_oauth_provider_oauth_id_key
    UNIQUE (oauth_provider, oauth_id);

-- Make OAuth fields required again (will fail if password-only users exist)
ALTER TABLE users
    ALTER COLUMN oauth_provider SET NOT NULL,
    ALTER COLUMN oauth_id SET NOT NULL;

-- Remove username and password_hash columns
ALTER TABLE users
    DROP COLUMN username,
    DROP COLUMN password_hash;
