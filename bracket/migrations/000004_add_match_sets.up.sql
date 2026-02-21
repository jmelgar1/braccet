-- Add match_sets table for set-based scoring
-- Each match can have multiple sets with individual scores

CREATE TABLE match_sets (
    id BIGSERIAL PRIMARY KEY,
    match_id BIGINT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    set_number INT NOT NULL,
    participant1_score INT NOT NULL DEFAULT 0,
    participant2_score INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (match_id, set_number),
    CHECK (set_number >= 1)
);

CREATE INDEX idx_match_sets_match_id ON match_sets(match_id);

-- Reuse existing trigger function for updated_at
CREATE TRIGGER update_match_sets_updated_at
    BEFORE UPDATE ON match_sets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Remove old score columns from matches (replaced by match_sets)
ALTER TABLE matches DROP COLUMN IF EXISTS participant1_score;
ALTER TABLE matches DROP COLUMN IF EXISTS participant2_score;
