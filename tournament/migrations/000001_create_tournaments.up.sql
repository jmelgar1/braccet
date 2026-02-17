-- Database: braccet_tournament
-- Note: organizer_id references Auth Service user, validated via API call

-- Create enum types
CREATE TYPE tournament_format AS ENUM ('single_elimination', 'double_elimination');
CREATE TYPE tournament_status AS ENUM ('draft', 'registration', 'in_progress', 'completed', 'cancelled');

CREATE TABLE tournaments (
    id BIGSERIAL PRIMARY KEY,
    organizer_id BIGINT NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    game VARCHAR(100),
    format tournament_format NOT NULL,
    status tournament_status DEFAULT 'draft',
    max_participants INT,
    registration_open BOOLEAN DEFAULT FALSE,
    settings JSONB,
    starts_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tournaments_status ON tournaments(status);
CREATE INDEX idx_tournaments_organizer ON tournaments(organizer_id);

-- Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_tournaments_updated_at
    BEFORE UPDATE ON tournaments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
