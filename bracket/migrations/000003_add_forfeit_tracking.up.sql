-- Add forfeit tracking to matches
-- forfeit_winner_id indicates the match was won by forfeit (opponent withdrew)
-- If NULL, match was played normally; if set, indicates the winner won by forfeit

ALTER TABLE matches ADD COLUMN forfeit_winner_id BIGINT;

-- Index for efficient lookup of forfeited matches
CREATE INDEX idx_matches_forfeit_winner ON matches(forfeit_winner_id) WHERE forfeit_winner_id IS NOT NULL;
