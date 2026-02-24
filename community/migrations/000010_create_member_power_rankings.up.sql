-- Current power rankings per member per system
CREATE TABLE member_power_rankings (
    id BIGSERIAL PRIMARY KEY,
    member_id BIGINT NOT NULL REFERENCES community_members(id) ON DELETE CASCADE,
    pr_system_id BIGINT NOT NULL REFERENCES power_ranking_systems(id) ON DELETE CASCADE,

    -- Current ranking
    total_points DECIMAL(10,2) NOT NULL DEFAULT 0,
    rank INT,                                    -- Current rank (1 = best)

    -- Component scores (for transparency)
    achievements_score DECIMAL(10,2) NOT NULL DEFAULT 0,
    form_score DECIMAL(10,2) NOT NULL DEFAULT 0,
    lan_score DECIMAL(10,2) NOT NULL DEFAULT 0,

    -- Form metrics (cached for display)
    form_wins INT NOT NULL DEFAULT 0,
    form_set_wins INT NOT NULL DEFAULT 0,
    form_set_losses INT NOT NULL DEFAULT 0,
    form_avg_opponent_rating INT,
    form_best_win_rating INT,                    -- Highest-rated opponent defeated

    -- LAN metrics (cached for display)
    lan_placements_count INT NOT NULL DEFAULT 0,

    last_calculated_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(member_id, pr_system_id)
);

CREATE INDEX idx_member_pr_ranking ON member_power_rankings(pr_system_id, total_points DESC);
CREATE INDEX idx_member_pr_member ON member_power_rankings(member_id);

-- Trigger to auto-update updated_at
CREATE TRIGGER update_member_power_rankings_updated_at
    BEFORE UPDATE ON member_power_rankings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE member_power_rankings IS 'Current power rankings with component breakdowns';
COMMENT ON COLUMN member_power_rankings.total_points IS 'Weighted sum of all component scores';
COMMENT ON COLUMN member_power_rankings.rank IS 'Current leaderboard position (1 = best)';
