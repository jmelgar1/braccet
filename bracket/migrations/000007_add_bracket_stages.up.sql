-- Create bracket_stages table to store stage-level settings (name and best-of)
CREATE TABLE bracket_stages (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL,
    bracket_type bracket_type DEFAULT 'winners',
    round INT NOT NULL,
    stage_name VARCHAR(100),
    best_of INT DEFAULT 1 CHECK (best_of IN (1, 3, 5, 7, 9)),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (tournament_id, bracket_type, round)
);

CREATE INDEX idx_bracket_stages_tournament ON bracket_stages(tournament_id);

-- Create trigger for updated_at auto-update
CREATE TRIGGER update_bracket_stages_updated_at
    BEFORE UPDATE ON bracket_stages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
