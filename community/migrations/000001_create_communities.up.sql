-- Communities table
-- Note: owner_id references Auth Service user, validated via API call (no FK)

CREATE TABLE communities (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(50) NOT NULL UNIQUE,
    owner_id BIGINT NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    game VARCHAR(100),
    avatar_url TEXT,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_communities_owner ON communities(owner_id);
CREATE INDEX idx_communities_slug ON communities(slug);
CREATE INDEX idx_communities_game ON communities(game);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_communities_updated_at
    BEFORE UPDATE ON communities
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
