DROP TRIGGER IF EXISTS update_community_members_updated_at ON community_members;
DROP INDEX IF EXISTS idx_community_members_elo;
DROP INDEX IF EXISTS idx_community_members_user;
DROP INDEX IF EXISTS idx_community_members_community;
DROP INDEX IF EXISTS idx_community_member_user;
DROP TABLE IF EXISTS community_members;
DROP TYPE IF EXISTS member_role;
