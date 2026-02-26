-- Rollback multi-tournament events support

DROP TABLE IF EXISTS event_advancement;
DROP TABLE IF EXISTS event_tournaments;
DROP TABLE IF EXISTS events;
DROP TYPE IF EXISTS event_tournament_role;
DROP TYPE IF EXISTS event_status;
