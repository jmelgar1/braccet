-- Remove participant icon URL columns from matches table
ALTER TABLE matches DROP COLUMN IF EXISTS participant1_icon_url;
ALTER TABLE matches DROP COLUMN IF EXISTS participant2_icon_url;
