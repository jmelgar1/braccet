-- Database: braccet_tournament
-- Note: user_id references Auth Service user, validated via API call

CREATE TYPE participant_status AS ENUM ('registered', 'checked_in', 'active', 'eliminated', 'disqualified');

CREATE TABLE participants (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    seed INT,
    status participant_status DEFAULT 'registered',
    checked_in_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    UNIQUE (tournament_id, user_id)
);

CREATE INDEX idx_participants_tournament_seed ON participants(tournament_id, seed);
