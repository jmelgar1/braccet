-- Multi-stage tournament support
-- Stores stage definitions for multi-stage tournaments with group stages and finals

-- Tournament stages configuration table
CREATE TABLE tournament_stages (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    stage_order INT NOT NULL,  -- 1, 2, 3 for group stages; 0 for final stage
    stage_type VARCHAR(20) NOT NULL CHECK (stage_type IN ('group', 'final')),
    format VARCHAR(30) NOT NULL CHECK (format IN ('single_elimination', 'double_elimination', 'swiss')),
    participants_per_group INT,  -- NULL for final stage
    advancing_per_group INT,     -- NULL for final stage
    swiss_rounds INT,            -- Optional override for Swiss format
    is_active BOOLEAN DEFAULT FALSE,
    is_complete BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (tournament_id, stage_order)
);

CREATE INDEX idx_tournament_stages_tournament ON tournament_stages(tournament_id);
CREATE INDEX idx_tournament_stages_active ON tournament_stages(tournament_id, is_active);

-- Groups within a stage (e.g., "Group A", "Group B")
CREATE TABLE stage_groups (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    stage_id BIGINT NOT NULL REFERENCES tournament_stages(id) ON DELETE CASCADE,
    group_name VARCHAR(50) NOT NULL,
    group_order INT NOT NULL,  -- For display ordering (0 = Group A, 1 = Group B, etc.)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (stage_id, group_order)
);

CREATE INDEX idx_stage_groups_stage ON stage_groups(stage_id);
CREATE INDEX idx_stage_groups_tournament ON stage_groups(tournament_id);

-- Group assignments - tracks which participants are in which groups
CREATE TABLE group_assignments (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    stage_id BIGINT NOT NULL REFERENCES tournament_stages(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES stage_groups(id) ON DELETE CASCADE,
    participant_id BIGINT NOT NULL,  -- References tournament participants
    seed_in_group INT,  -- Seed within the group (1-based)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (stage_id, participant_id)
);

CREATE INDEX idx_group_assignments_group ON group_assignments(group_id);
CREATE INDEX idx_group_assignments_participant ON group_assignments(participant_id);
CREATE INDEX idx_group_assignments_stage ON group_assignments(stage_id);

-- Ranking criteria configuration per stage (priority-ordered tiebreakers)
CREATE TYPE ranking_criterion AS ENUM (
    'match_wins',
    'set_wins',
    'set_win_pct',
    'set_differential',
    'points_scored',
    'points_differential'
);

CREATE TABLE stage_ranking_criteria (
    id BIGSERIAL PRIMARY KEY,
    stage_id BIGINT NOT NULL REFERENCES tournament_stages(id) ON DELETE CASCADE,
    criterion ranking_criterion NOT NULL,
    priority INT NOT NULL,  -- Lower = higher priority (1 is primary, 2 is secondary, etc.)

    UNIQUE (stage_id, priority),
    UNIQUE (stage_id, criterion)
);

CREATE INDEX idx_ranking_criteria_stage ON stage_ranking_criteria(stage_id);

-- Placement match configuration (for tiebreakers)
CREATE TABLE placement_match_config (
    id BIGSERIAL PRIMARY KEY,
    stage_id BIGINT NOT NULL REFERENCES tournament_stages(id) ON DELETE CASCADE,
    enabled BOOLEAN DEFAULT FALSE,
    depth INT DEFAULT 1,  -- How many positions to resolve via matches (e.g., 2 = resolve ties for positions 1 and 2)

    UNIQUE (stage_id)
);

-- Trigger for tournament_stages updated_at
CREATE TRIGGER update_tournament_stages_updated_at
    BEFORE UPDATE ON tournament_stages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
