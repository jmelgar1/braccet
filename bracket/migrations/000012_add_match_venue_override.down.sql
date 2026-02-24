-- Remove venue override from matches
DROP INDEX IF EXISTS idx_matches_venue;

ALTER TABLE matches
DROP COLUMN IF EXISTS venue_override;

DROP TYPE IF EXISTS match_venue_type;
