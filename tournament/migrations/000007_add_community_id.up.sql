-- Add community_id to tournaments
-- Note: community_id references Community Service, validated via API call (no FK)
ALTER TABLE tournaments ADD COLUMN community_id BIGINT;

CREATE INDEX idx_tournaments_community ON tournaments(community_id);
