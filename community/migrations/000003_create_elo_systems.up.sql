-- ELO systems allow communities to define multiple rating configurations
-- e.g., "1v1 Ranked", "Team Mode", "Casual"

CREATE TABLE elo_systems (
    id BIGSERIAL PRIMARY KEY,
    community_id BIGINT NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Core ELO configuration
    starting_rating INT NOT NULL DEFAULT 1000,
    k_factor INT NOT NULL DEFAULT 32,
    floor_rating INT NOT NULL DEFAULT 100,

    -- Provisional period (higher K-factor for new players)
    provisional_games INT NOT NULL DEFAULT 10,
    provisional_k_factor INT NOT NULL DEFAULT 64,

    -- Win streak bonuses
    win_streak_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    win_streak_threshold INT DEFAULT 3,
    win_streak_bonus INT DEFAULT 5,

    -- Rating decay (optional)
    decay_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    decay_days INT DEFAULT 30,
    decay_amount INT DEFAULT 10,
    decay_floor INT DEFAULT 800,

    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_elo_systems_community ON elo_systems(community_id);
CREATE INDEX idx_elo_systems_default ON elo_systems(community_id, is_default) WHERE is_default = TRUE;

-- Ensure only one default per community
CREATE UNIQUE INDEX idx_elo_systems_single_default
    ON elo_systems(community_id)
    WHERE is_default = TRUE;

CREATE TRIGGER update_elo_systems_updated_at
    BEFORE UPDATE ON elo_systems
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
