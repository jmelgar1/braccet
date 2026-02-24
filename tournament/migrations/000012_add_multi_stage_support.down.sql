-- Reverse multi-stage tournament support
DROP TRIGGER IF EXISTS update_tournament_stages_updated_at ON tournament_stages;
DROP TABLE IF EXISTS placement_match_config;
DROP TABLE IF EXISTS stage_ranking_criteria;
DROP TABLE IF EXISTS group_assignments;
DROP TABLE IF EXISTS stage_groups;
DROP TABLE IF EXISTS tournament_stages;
DROP TYPE IF EXISTS ranking_criterion;
