-- Tournament placement history for achievements component
CREATE TABLE tournament_placements (
    id BIGSERIAL PRIMARY KEY,
    member_id BIGINT NOT NULL REFERENCES community_members(id) ON DELETE CASCADE,
    pr_system_id BIGINT NOT NULL REFERENCES power_ranking_systems(id) ON DELETE CASCADE,
    tournament_id BIGINT NOT NULL,              -- References tournament service

    -- Placement details
    placement INT NOT NULL,                      -- Final placement (1, 2, 3, etc.)
    participant_count INT NOT NULL,              -- Total participants in tournament

    -- Tournament metadata (denormalized for calculation efficiency)
    tournament_class VARCHAR(20) NOT NULL,       -- major, world_lan, continental, regional, online
    prize_pool_usd DECIMAL(12,2),
    is_bo3_playoffs BOOLEAN DEFAULT FALSE,
    avg_opponent_rating INT,                     -- Average ELO of participants

    -- Venue tracking for LAN component
    is_lan BOOLEAN NOT NULL DEFAULT FALSE,

    -- Points (calculated at time of recording)
    raw_points INT NOT NULL,                     -- Base points before decay

    completed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(member_id, tournament_id)
);

CREATE INDEX idx_placements_member ON tournament_placements(member_id, completed_at DESC);
CREATE INDEX idx_placements_system ON tournament_placements(pr_system_id);
CREATE INDEX idx_placements_lan ON tournament_placements(member_id, is_lan, completed_at DESC);
CREATE INDEX idx_placements_tournament ON tournament_placements(tournament_id);

COMMENT ON TABLE tournament_placements IS 'Tournament placement history for power ranking achievements calculation';
COMMENT ON COLUMN tournament_placements.raw_points IS 'Base points calculated at recording time (before time decay)';
COMMENT ON COLUMN tournament_placements.tournament_class IS 'Tournament class at time of completion (major, world_lan, continental, regional, online)';
