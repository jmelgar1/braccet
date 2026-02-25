-- Remove group-based index
DROP INDEX IF EXISTS idx_bracket_stages_group;

-- Remove the new unique index
DROP INDEX IF EXISTS bracket_stages_unique_idx;

-- Restore original unique constraint
ALTER TABLE bracket_stages ADD CONSTRAINT bracket_stages_tournament_id_bracket_type_round_key UNIQUE (tournament_id, bracket_type, round);

-- Remove the new columns
ALTER TABLE bracket_stages DROP COLUMN IF EXISTS group_id;
ALTER TABLE bracket_stages DROP COLUMN IF EXISTS stage_id;
