-- Drop triggers first
DROP TRIGGER IF EXISTS update_swiss_config_updated_at ON swiss_config;
DROP TRIGGER IF EXISTS update_swiss_standings_updated_at ON swiss_standings;

-- Drop tables
DROP TABLE IF EXISTS swiss_config;
DROP TABLE IF EXISTS swiss_pairing_history;
DROP TABLE IF EXISTS swiss_standings;
