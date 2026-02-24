DROP TRIGGER IF EXISTS update_stage_participant_pool_updated_at ON stage_participant_pool;
DROP INDEX IF EXISTS idx_stage_participant_pool_tournament;
DROP INDEX IF EXISTS idx_stage_participant_pool_stage;
DROP TABLE IF EXISTS stage_participant_pool;
