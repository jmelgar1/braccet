-- Remove draft status from tournaments
-- First, delete any tournaments in draft status (as agreed with user)
DELETE FROM tournaments WHERE status = 'draft';

-- Drop the default first (required before changing enum type)
ALTER TABLE tournaments ALTER COLUMN status DROP DEFAULT;

-- PostgreSQL doesn't allow removing enum values directly
-- We need to: create new type, alter column, drop old type, rename new type
CREATE TYPE tournament_status_new AS ENUM ('registration', 'in_progress', 'completed', 'cancelled');

ALTER TABLE tournaments
    ALTER COLUMN status TYPE tournament_status_new
    USING status::text::tournament_status_new;

DROP TYPE tournament_status;

ALTER TYPE tournament_status_new RENAME TO tournament_status;

-- Set the new default
ALTER TABLE tournaments ALTER COLUMN status SET DEFAULT 'registration';
