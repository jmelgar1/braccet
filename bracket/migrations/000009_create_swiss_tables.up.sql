-- Swiss standings track cumulative results per participant
CREATE TABLE swiss_standings (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL,
    participant_id BIGINT NOT NULL,
    participant_name VARCHAR(100),
    participant_icon_url VARCHAR(500),
    seed INT NOT NULL,
    wins INT DEFAULT 0,
    losses INT DEFAULT 0,
    game_wins INT DEFAULT 0,
    game_losses INT DEFAULT 0,
    opponent_wins INT DEFAULT 0,  -- Buchholz tiebreaker: sum of opponents' wins
    has_bye BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (tournament_id, participant_id)
);

CREATE INDEX idx_swiss_standings_tournament ON swiss_standings(tournament_id);
CREATE INDEX idx_swiss_standings_ranking ON swiss_standings(tournament_id, wins DESC, game_wins DESC);

-- Swiss pairing history prevents rematches
CREATE TABLE swiss_pairing_history (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL,
    participant1_id BIGINT NOT NULL,
    participant2_id BIGINT NOT NULL,
    round INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (tournament_id, participant1_id, participant2_id)
);

CREATE INDEX idx_swiss_history_tournament ON swiss_pairing_history(tournament_id);
CREATE INDEX idx_swiss_history_lookup ON swiss_pairing_history(tournament_id, participant1_id);

-- Swiss config stores tournament-level Swiss settings
CREATE TABLE swiss_config (
    tournament_id BIGINT PRIMARY KEY,
    total_rounds INT NOT NULL,
    current_round INT DEFAULT 0,
    is_complete BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Trigger to update updated_at timestamp
CREATE TRIGGER update_swiss_standings_updated_at
    BEFORE UPDATE ON swiss_standings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_swiss_config_updated_at
    BEFORE UPDATE ON swiss_config
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
