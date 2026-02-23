-- Add icon_url column to community_members for member avatars/icons
ALTER TABLE community_members
ADD COLUMN icon_url VARCHAR(512);

COMMENT ON COLUMN community_members.icon_url IS 'URL to member avatar/icon image stored in R2';
