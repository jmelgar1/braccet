-- Add region column to community_members for geographic categorization
ALTER TABLE community_members
ADD COLUMN region VARCHAR(2);

COMMENT ON COLUMN community_members.region IS 'Two-letter region code (organizer-defined)';
