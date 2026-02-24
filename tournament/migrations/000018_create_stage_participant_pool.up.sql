-- Stage participant pool - tracks which stage each participant starts in
-- Participants in later stages (e.g., Finals) wait idle until their stage becomes active

CREATE TABLE stage_participant_pool (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    stage_id BIGINT NOT NULL REFERENCES tournament_stages(id) ON DELETE CASCADE,
    participant_id BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Each participant can only be in one stage's pool
    UNIQUE (tournament_id, participant_id)
);

-- Index for efficient queries by stage
CREATE INDEX idx_stage_participant_pool_stage ON stage_participant_pool(stage_id);

-- Index for efficient queries by tournament
CREATE INDEX idx_stage_participant_pool_tournament ON stage_participant_pool(tournament_id);

-- Trigger for updated_at auto-update
CREATE TRIGGER update_stage_participant_pool_updated_at
    BEFORE UPDATE ON stage_participant_pool
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
