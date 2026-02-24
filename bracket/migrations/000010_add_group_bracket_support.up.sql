-- Group bracket support for multi-stage tournaments
-- Adds stage/group tracking to matches and standings tables

-- Group standings - tracks individual participant stats within a group
CREATE TABLE group_standings (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL,
    stage_id BIGINT NOT NULL,      -- References tournament_stages in tournament service
    group_id BIGINT NOT NULL,      -- References stage_groups in tournament service
    participant_id BIGINT NOT NULL,
    participant_name VARCHAR(100),
    participant_icon_url VARCHAR(500),
    seed INT,

    -- Match stats
    match_wins INT DEFAULT 0,
    match_losses INT DEFAULT 0,

    -- Set/game stats (for set-based scoring)
    set_wins INT DEFAULT 0,
    set_losses INT DEFAULT 0,

    -- Point stats (aggregate of all set scores)
    points_scored INT DEFAULT 0,
    points_against INT DEFAULT 0,

    -- Tiebreaker stats
    opponent_match_wins INT DEFAULT 0,  -- Buchholz-style tiebreaker

    -- Final ranking within group (computed after group completes)
    group_rank INT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (tournament_id, stage_id, group_id, participant_id)
);

CREATE INDEX idx_group_standings_tournament ON group_standings(tournament_id);
CREATE INDEX idx_group_standings_stage ON group_standings(stage_id);
CREATE INDEX idx_group_standings_group ON group_standings(group_id);
CREATE INDEX idx_group_standings_participant ON group_standings(participant_id);

-- Stage standings - cross-group aggregation for determining advancement
CREATE TABLE stage_standings (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL,
    stage_id BIGINT NOT NULL,
    participant_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    group_rank INT NOT NULL,  -- Rank within their group

    -- Aggregated stats for cross-group comparison
    match_wins INT DEFAULT 0,
    match_losses INT DEFAULT 0,
    set_wins INT DEFAULT 0,
    set_losses INT DEFAULT 0,
    points_scored INT DEFAULT 0,
    points_against INT DEFAULT 0,

    -- Final stage rank (computed after all groups complete)
    stage_rank INT,
    advances BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (tournament_id, stage_id, participant_id)
);

CREATE INDEX idx_stage_standings_tournament ON stage_standings(tournament_id);
CREATE INDEX idx_stage_standings_stage ON stage_standings(stage_id);
CREATE INDEX idx_stage_standings_advances ON stage_standings(stage_id, advances);

-- Add stage_id and group_id to matches table for scoping
ALTER TABLE matches ADD COLUMN stage_id BIGINT;
ALTER TABLE matches ADD COLUMN group_id BIGINT;

CREATE INDEX idx_matches_stage ON matches(tournament_id, stage_id);
CREATE INDEX idx_matches_group ON matches(tournament_id, group_id);

-- Add stage_id and group_id to swiss tables for multi-stage Swiss groups
ALTER TABLE swiss_standings ADD COLUMN stage_id BIGINT;
ALTER TABLE swiss_standings ADD COLUMN group_id BIGINT;

ALTER TABLE swiss_config ADD COLUMN stage_id BIGINT;
ALTER TABLE swiss_config ADD COLUMN group_id BIGINT;

ALTER TABLE swiss_pairing_history ADD COLUMN stage_id BIGINT;
ALTER TABLE swiss_pairing_history ADD COLUMN group_id BIGINT;

-- Create indexes for the new columns
CREATE INDEX idx_swiss_standings_stage ON swiss_standings(tournament_id, stage_id);
CREATE INDEX idx_swiss_standings_group ON swiss_standings(tournament_id, group_id);
CREATE INDEX idx_swiss_config_stage ON swiss_config(tournament_id, stage_id);
CREATE INDEX idx_swiss_config_group ON swiss_config(tournament_id, group_id);

-- Add stage_id and group_id to bracket_stages table
ALTER TABLE bracket_stages ADD COLUMN stage_id BIGINT;
ALTER TABLE bracket_stages ADD COLUMN group_id BIGINT;

CREATE INDEX idx_bracket_stages_stage ON bracket_stages(tournament_id, stage_id);
CREATE INDEX idx_bracket_stages_group ON bracket_stages(tournament_id, group_id);

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_group_standings_updated_at
    BEFORE UPDATE ON group_standings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_stage_standings_updated_at
    BEFORE UPDATE ON stage_standings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
