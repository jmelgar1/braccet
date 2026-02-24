-- Remove venue type from tournament stages
DROP INDEX IF EXISTS idx_tournament_stages_venue;

ALTER TABLE tournament_stages
DROP COLUMN IF EXISTS venue_type;

DROP TYPE IF EXISTS venue_type;
