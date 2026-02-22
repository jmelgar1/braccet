DROP INDEX IF EXISTS idx_tournaments_elo_system;
ALTER TABLE tournaments DROP COLUMN IF EXISTS elo_system_id;
