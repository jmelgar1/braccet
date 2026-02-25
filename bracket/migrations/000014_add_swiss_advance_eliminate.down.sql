-- Remove index
DROP INDEX IF EXISTS idx_swiss_standings_status;

-- Remove columns from swiss_standings
ALTER TABLE swiss_standings
DROP COLUMN IF EXISTS match_wins,
DROP COLUMN IF EXISTS status;

-- Remove enum type
DROP TYPE IF EXISTS swiss_participant_status;

-- Remove columns from swiss_config
ALTER TABLE swiss_config
DROP COLUMN IF EXISTS losses_to_eliminate,
DROP COLUMN IF EXISTS wins_to_advance;
