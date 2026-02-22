-- Community members can be:
-- 1. Linked to a real user (user_id NOT NULL) - actual platform users
-- 2. Ghost members (user_id NULL) - created by organizer, just have display_name

CREATE TYPE member_role AS ENUM ('owner', 'admin', 'member');

CREATE TABLE community_members (
    id BIGSERIAL PRIMARY KEY,
    community_id BIGINT NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    user_id BIGINT,
    display_name VARCHAR(100) NOT NULL,
    role member_role DEFAULT 'member',

    -- Future ELO/ranking fields (nullable for now)
    elo_rating INT,
    ranking_points INT,
    matches_played INT DEFAULT 0,
    matches_won INT DEFAULT 0,

    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Partial unique index: real users can only join once per community
CREATE UNIQUE INDEX idx_community_member_user
    ON community_members(community_id, user_id)
    WHERE user_id IS NOT NULL;

CREATE INDEX idx_community_members_community ON community_members(community_id);
CREATE INDEX idx_community_members_user ON community_members(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_community_members_elo ON community_members(community_id, elo_rating DESC);

CREATE TRIGGER update_community_members_updated_at
    BEFORE UPDATE ON community_members
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
