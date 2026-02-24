-- Reverse group bracket support

-- Drop triggers
DROP TRIGGER IF EXISTS update_stage_standings_updated_at ON stage_standings;
DROP TRIGGER IF EXISTS update_group_standings_updated_at ON group_standings;

-- Drop indexes on bracket_stages
DROP INDEX IF EXISTS idx_bracket_stages_group;
DROP INDEX IF EXISTS idx_bracket_stages_stage;

-- Remove columns from bracket_stages
ALTER TABLE bracket_stages DROP COLUMN IF EXISTS group_id;
ALTER TABLE bracket_stages DROP COLUMN IF EXISTS stage_id;

-- Drop indexes on swiss tables
DROP INDEX IF EXISTS idx_swiss_config_group;
DROP INDEX IF EXISTS idx_swiss_config_stage;
DROP INDEX IF EXISTS idx_swiss_standings_group;
DROP INDEX IF EXISTS idx_swiss_standings_stage;

-- Remove columns from swiss tables
ALTER TABLE swiss_pairing_history DROP COLUMN IF EXISTS group_id;
ALTER TABLE swiss_pairing_history DROP COLUMN IF EXISTS stage_id;
ALTER TABLE swiss_config DROP COLUMN IF EXISTS group_id;
ALTER TABLE swiss_config DROP COLUMN IF EXISTS stage_id;
ALTER TABLE swiss_standings DROP COLUMN IF EXISTS group_id;
ALTER TABLE swiss_standings DROP COLUMN IF EXISTS stage_id;

-- Drop indexes on matches
DROP INDEX IF EXISTS idx_matches_group;
DROP INDEX IF EXISTS idx_matches_stage;

-- Remove columns from matches
ALTER TABLE matches DROP COLUMN IF EXISTS group_id;
ALTER TABLE matches DROP COLUMN IF EXISTS stage_id;

-- Drop standings tables
DROP TABLE IF EXISTS stage_standings;
DROP TABLE IF EXISTS group_standings;
