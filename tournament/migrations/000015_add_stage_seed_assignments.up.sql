-- Manual seed assignments for multi-stage tournaments
-- Stores organizer-selected seeds BEFORE a stage starts
-- These are used to influence serpentine distribution when StartStage is called

CREATE TABLE stage_seed_assignments (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    stage_id BIGINT NOT NULL REFERENCES tournament_stages(id) ON DELETE CASCADE,
    participant_id BIGINT NOT NULL,  -- References tournament participants
    target_group_order INT NOT NULL, -- Which group (0 = Group A, 1 = Group B, etc.)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Each participant can only be seeded to one group per stage
    UNIQUE (stage_id, participant_id)
);

CREATE INDEX idx_stage_seed_assignments_stage ON stage_seed_assignments(stage_id);
CREATE INDEX idx_stage_seed_assignments_tournament ON stage_seed_assignments(tournament_id);

-- Trigger for updated_at
CREATE TRIGGER update_stage_seed_assignments_updated_at
    BEFORE UPDATE ON stage_seed_assignments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
