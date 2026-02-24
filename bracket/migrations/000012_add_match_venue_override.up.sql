-- Match-level venue override for hybrid stages
-- NULL means inherit from stage, 'lan' or 'online' overrides the stage setting
CREATE TYPE match_venue_type AS ENUM ('lan', 'online');

ALTER TABLE matches
ADD COLUMN venue_override match_venue_type;

-- Index for queries filtering by venue
CREATE INDEX idx_matches_venue ON matches(venue_override) WHERE venue_override IS NOT NULL;

COMMENT ON COLUMN matches.venue_override IS 'Overrides stage venue type for hybrid stages (NULL = inherit from stage)';
