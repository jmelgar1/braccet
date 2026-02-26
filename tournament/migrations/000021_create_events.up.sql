-- Multi-tournament Events support
-- Events are containers for related tournaments (qualifiers -> main event)
-- Events are community-bound (community_id is required)

-- Event status enum
CREATE TYPE event_status AS ENUM ('draft', 'registration', 'in_progress', 'completed', 'cancelled');

-- Event tournament role enum
CREATE TYPE event_tournament_role AS ENUM ('qualifier', 'main');

-- Events table - the container for related tournaments
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(50) UNIQUE NOT NULL,
    community_id BIGINT NOT NULL,  -- Required: events are community-bound
    organizer_id BIGINT NOT NULL,  -- References auth service user
    name VARCHAR(255) NOT NULL,
    description TEXT,
    game VARCHAR(100),
    status event_status DEFAULT 'draft',
    starts_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_events_community ON events(community_id);
CREATE INDEX idx_events_organizer ON events(organizer_id);
CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_events_slug ON events(slug);

-- Trigger for events updated_at
CREATE TRIGGER update_events_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Event tournaments - links tournaments to events with roles and progression rules
CREATE TABLE event_tournaments (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    role event_tournament_role NOT NULL,
    display_order INT NOT NULL DEFAULT 0,  -- For UI ordering (qualifiers first, then main)
    advancing_count INT,  -- How many advance from this tournament (NULL for main event)
    target_tournament_id BIGINT REFERENCES tournaments(id) ON DELETE SET NULL,  -- Where winners go
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (event_id, tournament_id),
    UNIQUE (tournament_id)  -- A tournament can only belong to one event
);

CREATE INDEX idx_event_tournaments_event ON event_tournaments(event_id);
CREATE INDEX idx_event_tournaments_tournament ON event_tournaments(tournament_id);
CREATE INDEX idx_event_tournaments_target ON event_tournaments(target_tournament_id);

-- Event advancement - tracks participant lineage across tournaments
CREATE TABLE event_advancement (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    source_tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    source_participant_id BIGINT NOT NULL,  -- Participant ID in source tournament
    target_tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    target_participant_id BIGINT NOT NULL,  -- New participant ID in target tournament
    final_placement INT NOT NULL,  -- 1st, 2nd, etc. in source tournament
    advanced_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (source_tournament_id, source_participant_id),  -- Each source participant advances once
    UNIQUE (target_tournament_id, target_participant_id)   -- Each target participant has one origin
);

CREATE INDEX idx_event_advancement_event ON event_advancement(event_id);
CREATE INDEX idx_event_advancement_source ON event_advancement(source_tournament_id);
CREATE INDEX idx_event_advancement_target ON event_advancement(target_tournament_id);
