-- Full history of rating changes for audit and display

CREATE TYPE elo_change_type AS ENUM ('match', 'decay', 'adjustment', 'initial');

CREATE TABLE elo_history (
    id BIGSERIAL PRIMARY KEY,
    member_id BIGINT NOT NULL REFERENCES community_members(id) ON DELETE CASCADE,
    elo_system_id BIGINT NOT NULL REFERENCES elo_systems(id) ON DELETE CASCADE,

    change_type elo_change_type NOT NULL,
    rating_before INT NOT NULL,
    rating_change INT NOT NULL,
    rating_after INT NOT NULL,

    -- Match context (NULL for non-match changes)
    match_id BIGINT,
    tournament_id BIGINT,
    opponent_member_id BIGINT REFERENCES community_members(id) ON DELETE SET NULL,
    opponent_rating_before INT,
    is_winner BOOLEAN,

    -- Calculation details for transparency
    k_factor_used INT,
    expected_score DECIMAL(5,4),
    win_streak_bonus INT DEFAULT 0,

    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_elo_history_member ON elo_history(member_id, created_at DESC);
CREATE INDEX idx_elo_history_system ON elo_history(elo_system_id, created_at DESC);
CREATE INDEX idx_elo_history_match ON elo_history(match_id) WHERE match_id IS NOT NULL;
CREATE INDEX idx_elo_history_tournament ON elo_history(tournament_id) WHERE tournament_id IS NOT NULL;
