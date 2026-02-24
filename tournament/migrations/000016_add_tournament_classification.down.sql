-- Remove classification columns from tournaments
DROP INDEX IF EXISTS idx_tournaments_class;

ALTER TABLE tournaments
DROP COLUMN IF EXISTS tournament_class,
DROP COLUMN IF EXISTS prize_pool_usd;

DROP TYPE IF EXISTS tournament_class;
