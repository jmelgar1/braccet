DROP INDEX IF EXISTS idx_participants_community_member;
ALTER TABLE participants DROP COLUMN community_member_id;
