DROP TRIGGER IF EXISTS update_matches_updated_at ON matches;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS matches;
DROP TYPE IF EXISTS match_status;
DROP TYPE IF EXISTS bracket_type;
