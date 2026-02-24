DROP TRIGGER IF EXISTS update_member_power_rankings_updated_at ON member_power_rankings;
DROP INDEX IF EXISTS idx_member_pr_member;
DROP INDEX IF EXISTS idx_member_pr_ranking;
DROP TABLE IF EXISTS member_power_rankings;
