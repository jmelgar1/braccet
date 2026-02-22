DROP TRIGGER IF EXISTS update_member_elo_ratings_updated_at ON member_elo_ratings;
DROP INDEX IF EXISTS idx_member_elo_ratings_leaderboard;
DROP INDEX IF EXISTS idx_member_elo_ratings_system;
DROP INDEX IF EXISTS idx_member_elo_ratings_member;
DROP TABLE IF EXISTS member_elo_ratings;
