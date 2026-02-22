DROP TRIGGER IF EXISTS update_elo_systems_updated_at ON elo_systems;
DROP INDEX IF EXISTS idx_elo_systems_single_default;
DROP INDEX IF EXISTS idx_elo_systems_default;
DROP INDEX IF EXISTS idx_elo_systems_community;
DROP TABLE IF EXISTS elo_systems;
