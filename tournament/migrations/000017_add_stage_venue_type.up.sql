-- Venue type for tournament stages (LAN/Online/Hybrid tracking)
CREATE TYPE venue_type AS ENUM ('lan', 'online', 'hybrid');

-- Add venue type to tournament stages
ALTER TABLE tournament_stages
ADD COLUMN venue_type venue_type NOT NULL DEFAULT 'online';

-- Index for filtering by venue type
CREATE INDEX idx_tournament_stages_venue ON tournament_stages(venue_type);

COMMENT ON COLUMN tournament_stages.venue_type IS 'Venue type for power ranking LAN tracking (lan, online, hybrid)';
