DROP TRIGGER IF EXISTS update_power_ranking_systems_updated_at ON power_ranking_systems;
DROP INDEX IF EXISTS idx_pr_systems_single_default;
DROP INDEX IF EXISTS idx_pr_systems_community;
DROP TABLE IF EXISTS power_ranking_systems;
