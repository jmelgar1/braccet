-- Power Ranking Systems (configurable per community, parallel to ELO systems)
CREATE TABLE power_ranking_systems (
    id BIGSERIAL PRIMARY KEY,
    community_id BIGINT NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Component weights (must sum to 100)
    achievements_weight INT NOT NULL DEFAULT 50,
    form_weight INT NOT NULL DEFAULT 30,
    lan_weight INT NOT NULL DEFAULT 20,

    -- Achievements configuration
    achievement_decay_months INT NOT NULL DEFAULT 4,        -- ~50% decay after this many months
    achievement_window_months INT NOT NULL DEFAULT 12,      -- Rolling window in months

    -- Form configuration
    form_window_months INT NOT NULL DEFAULT 3,              -- Recent performance window

    -- LAN configuration
    lan_results_count INT NOT NULL DEFAULT 10,              -- Last N LAN results to consider

    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT weights_sum_100 CHECK (achievements_weight + form_weight + lan_weight = 100)
);

CREATE INDEX idx_pr_systems_community ON power_ranking_systems(community_id);
CREATE UNIQUE INDEX idx_pr_systems_single_default ON power_ranking_systems(community_id) WHERE is_default = TRUE;

-- Trigger to auto-update updated_at
CREATE TRIGGER update_power_ranking_systems_updated_at
    BEFORE UPDATE ON power_ranking_systems
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE power_ranking_systems IS 'Power ranking system configurations per community';
COMMENT ON COLUMN power_ranking_systems.achievements_weight IS 'Weight for tournament placements component (0-100)';
COMMENT ON COLUMN power_ranking_systems.form_weight IS 'Weight for recent form component (0-100)';
COMMENT ON COLUMN power_ranking_systems.lan_weight IS 'Weight for LAN results component (0-100)';
