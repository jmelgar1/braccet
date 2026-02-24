-- Match results cache for form calculation
-- Stores recent match outcomes with opponent ratings
CREATE TABLE power_ranking_match_cache (
    id BIGSERIAL PRIMARY KEY,
    member_id BIGINT NOT NULL REFERENCES community_members(id) ON DELETE CASCADE,
    pr_system_id BIGINT NOT NULL REFERENCES power_ranking_systems(id) ON DELETE CASCADE,
    match_id BIGINT NOT NULL,
    tournament_id BIGINT NOT NULL,

    is_winner BOOLEAN NOT NULL,
    sets_won INT NOT NULL DEFAULT 0,
    sets_lost INT NOT NULL DEFAULT 0,
    opponent_member_id BIGINT REFERENCES community_members(id) ON DELETE SET NULL,
    opponent_rating INT,                         -- Opponent ELO at time of match

    -- Venue tracking
    is_lan BOOLEAN NOT NULL DEFAULT FALSE,

    completed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(member_id, match_id)
);

CREATE INDEX idx_pr_match_cache_member ON power_ranking_match_cache(member_id, completed_at DESC);
CREATE INDEX idx_pr_match_cache_form ON power_ranking_match_cache(member_id, pr_system_id, completed_at DESC);
CREATE INDEX idx_pr_match_cache_tournament ON power_ranking_match_cache(tournament_id);

COMMENT ON TABLE power_ranking_match_cache IS 'Match results for power ranking form calculation';
COMMENT ON COLUMN power_ranking_match_cache.opponent_rating IS 'Opponent ELO rating at the time of the match';
