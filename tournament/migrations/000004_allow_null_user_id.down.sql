-- Revert: Remove partial index and restore NOT NULL + unique constraint

-- Drop the partial unique index
DROP INDEX IF EXISTS idx_participants_tournament_user;

-- Remove any participants without user_id (data loss - required for NOT NULL)
DELETE FROM participants WHERE user_id IS NULL;

-- Restore NOT NULL constraint
ALTER TABLE participants ALTER COLUMN user_id SET NOT NULL;

-- Restore the original unique constraint
ALTER TABLE participants ADD CONSTRAINT participants_tournament_id_user_id_key
    UNIQUE (tournament_id, user_id);
