-- Restore draft status to tournaments
-- Drop the default first
ALTER TABLE tournaments ALTER COLUMN status DROP DEFAULT;

-- Recreate enum with draft
CREATE TYPE tournament_status_new AS ENUM ('draft', 'registration', 'in_progress', 'completed', 'cancelled');

ALTER TABLE tournaments
    ALTER COLUMN status TYPE tournament_status_new
    USING status::text::tournament_status_new;

DROP TYPE tournament_status;

ALTER TYPE tournament_status_new RENAME TO tournament_status;

-- Change default back to draft
ALTER TABLE tournaments ALTER COLUMN status SET DEFAULT 'draft';
