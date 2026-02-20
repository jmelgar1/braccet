-- Remove slug column
DROP INDEX IF EXISTS idx_tournaments_slug;
ALTER TABLE tournaments DROP COLUMN IF EXISTS slug;
