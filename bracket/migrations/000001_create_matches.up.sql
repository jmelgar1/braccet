-- Database: braccet_bracket
-- Note: tournament_id and participant IDs are external references

CREATE TYPE bracket_type AS ENUM ('winners', 'losers', 'grand_final');
CREATE TYPE match_status AS ENUM ('pending', 'ready', 'in_progress', 'completed');

CREATE TABLE matches (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL,
    bracket_type bracket_type DEFAULT 'winners',
    round INT NOT NULL,
    position INT NOT NULL,
    participant1_id BIGINT,
    participant2_id BIGINT,
    participant1_name VARCHAR(100),
    participant2_name VARCHAR(100),
    winner_id BIGINT,
    participant1_score INT,
    participant2_score INT,
    status match_status DEFAULT 'pending',
    scheduled_at TIMESTAMP,
    completed_at TIMESTAMP,
    next_match_id BIGINT,
    loser_match_id BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (next_match_id) REFERENCES matches(id) ON DELETE SET NULL,
    FOREIGN KEY (loser_match_id) REFERENCES matches(id) ON DELETE SET NULL,
    UNIQUE (tournament_id, bracket_type, round, position)
);

CREATE INDEX idx_matches_tournament_round ON matches(tournament_id, bracket_type, round);

-- Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_matches_updated_at
    BEFORE UPDATE ON matches
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
