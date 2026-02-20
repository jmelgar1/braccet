-- Allow NULL user_id for display-name-only participants (non-registered users)
ALTER TABLE participants ALTER COLUMN user_id DROP NOT NULL;

-- Drop the existing unique constraint (handles both named and unnamed constraints)
ALTER TABLE participants DROP CONSTRAINT IF EXISTS participants_tournament_id_user_id_key;

-- Create a partial unique index that only applies when user_id is NOT NULL
-- This allows multiple display-name-only participants while preventing duplicate user registrations
CREATE UNIQUE INDEX idx_participants_tournament_user
    ON participants(tournament_id, user_id) WHERE user_id IS NOT NULL;
