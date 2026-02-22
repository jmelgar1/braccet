-- Link participants to community members for reusability
-- Note: community_member_id references Community Service, validated via API call (no FK)
ALTER TABLE participants ADD COLUMN community_member_id BIGINT;

CREATE INDEX idx_participants_community_member ON participants(community_member_id);
