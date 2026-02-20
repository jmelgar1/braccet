-- Fix OAuth unique constraint to allow multiple email/password users (NULL oauth fields)
-- The previous constraint with NULLS NOT DISTINCT prevented this

-- Drop the problematic constraint
ALTER TABLE users DROP CONSTRAINT users_oauth_unique;

-- Create a partial unique index that only applies when oauth fields are NOT NULL
CREATE UNIQUE INDEX users_oauth_unique ON users (oauth_provider, oauth_id)
WHERE oauth_provider IS NOT NULL AND oauth_id IS NOT NULL;
