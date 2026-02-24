-- Rollback tiebreaker support
-- Note: Cannot remove enum value 'tiebreaker' from bracket_type in PostgreSQL

DROP TRIGGER IF EXISTS update_swiss_tiebreakers_updated_at ON swiss_tiebreakers;
DROP TABLE IF EXISTS swiss_tiebreaker_results;
DROP TABLE IF EXISTS swiss_tiebreakers;
ALTER TABLE swiss_config DROP COLUMN IF EXISTS has_tiebreakers;
ALTER TABLE swiss_config DROP COLUMN IF EXISTS tiebreakers_complete;
