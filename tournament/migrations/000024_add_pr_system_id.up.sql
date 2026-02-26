-- Add pr_system_id to tournaments table
-- References community service power_ranking_systems table (cross-service reference)
ALTER TABLE tournaments ADD COLUMN pr_system_id BIGINT;
