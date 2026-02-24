-- Tournament classification for power ranking weighting
CREATE TYPE tournament_class AS ENUM (
    'major',           -- Premier events (e.g., World Championship)
    'world_lan',       -- World-level LAN events
    'continental',     -- Regional majors (e.g., European Championship)
    'regional',        -- Local/regional events
    'online'           -- Online tournaments
);

-- Add classification columns to tournaments
ALTER TABLE tournaments
ADD COLUMN tournament_class tournament_class,
ADD COLUMN prize_pool_usd DECIMAL(12,2);

-- Index for filtering by class
CREATE INDEX idx_tournaments_class ON tournaments(tournament_class) WHERE tournament_class IS NOT NULL;

COMMENT ON COLUMN tournaments.tournament_class IS 'Classification for power ranking weighting (can be auto-assigned or manually set)';
COMMENT ON COLUMN tournaments.prize_pool_usd IS 'Prize pool in USD for power ranking calculations';
