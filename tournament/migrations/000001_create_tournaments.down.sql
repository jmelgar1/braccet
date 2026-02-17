DROP TRIGGER IF EXISTS update_tournaments_updated_at ON tournaments;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS tournaments;
DROP TYPE IF EXISTS tournament_status;
DROP TYPE IF EXISTS tournament_format;
