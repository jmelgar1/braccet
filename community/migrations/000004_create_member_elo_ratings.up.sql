-- Per-system ratings for each community member
-- A member can have ratings in multiple ELO systems

CREATE TABLE member_elo_ratings (
    id BIGSERIAL PRIMARY KEY,
    member_id BIGINT NOT NULL REFERENCES community_members(id) ON DELETE CASCADE,
    elo_system_id BIGINT NOT NULL REFERENCES elo_systems(id) ON DELETE CASCADE,

    rating INT NOT NULL,
    games_played INT NOT NULL DEFAULT 0,
    games_won INT NOT NULL DEFAULT 0,
    current_win_streak INT NOT NULL DEFAULT 0,
    highest_rating INT NOT NULL,
    lowest_rating INT NOT NULL,

    last_game_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(member_id, elo_system_id)
);

CREATE INDEX idx_member_elo_ratings_member ON member_elo_ratings(member_id);
CREATE INDEX idx_member_elo_ratings_system ON member_elo_ratings(elo_system_id);
CREATE INDEX idx_member_elo_ratings_leaderboard ON member_elo_ratings(elo_system_id, rating DESC);

CREATE TRIGGER update_member_elo_ratings_updated_at
    BEFORE UPDATE ON member_elo_ratings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
