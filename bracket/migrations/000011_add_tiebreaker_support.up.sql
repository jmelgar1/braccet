-- Swiss tiebreaker support
-- When all ranking criteria are tied, create tiebreaker matches to resolve positions

-- Add tiebreaker bracket type
ALTER TYPE bracket_type ADD VALUE 'tiebreaker';

-- Track tiebreaker brackets (mini brackets for resolving ties)
CREATE TABLE swiss_tiebreakers (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL,
    tie_position INT NOT NULL,  -- The rank position being resolved (1 = tied for 1st)
    participant_ids BIGINT[] NOT NULL,  -- Array of participant IDs in this tie
    is_complete BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (tournament_id, tie_position)
);

CREATE INDEX idx_swiss_tiebreakers_tournament ON swiss_tiebreakers(tournament_id);

-- Track resolved positions from tiebreakers
CREATE TABLE swiss_tiebreaker_results (
    id BIGSERIAL PRIMARY KEY,
    tiebreaker_id BIGINT NOT NULL REFERENCES swiss_tiebreakers(id) ON DELETE CASCADE,
    participant_id BIGINT NOT NULL,
    final_position INT NOT NULL,  -- Their resolved rank (relative to tie_position)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (tiebreaker_id, participant_id)
);

CREATE INDEX idx_tiebreaker_results_tiebreaker ON swiss_tiebreaker_results(tiebreaker_id);

-- Add tiebreaker tracking columns to swiss_config
ALTER TABLE swiss_config ADD COLUMN has_tiebreakers BOOLEAN DEFAULT FALSE;
ALTER TABLE swiss_config ADD COLUMN tiebreakers_complete BOOLEAN DEFAULT FALSE;

-- Trigger for updated_at on swiss_tiebreakers
CREATE TRIGGER update_swiss_tiebreakers_updated_at
    BEFORE UPDATE ON swiss_tiebreakers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
