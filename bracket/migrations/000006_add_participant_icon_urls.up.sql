-- Add participant icon URL columns to matches table
ALTER TABLE matches ADD COLUMN participant1_icon_url TEXT;
ALTER TABLE matches ADD COLUMN participant2_icon_url TEXT;
