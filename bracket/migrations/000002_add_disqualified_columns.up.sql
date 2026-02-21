-- Add disqualified tracking to matches
ALTER TABLE matches ADD COLUMN participant1_disqualified BOOLEAN DEFAULT FALSE;
ALTER TABLE matches ADD COLUMN participant2_disqualified BOOLEAN DEFAULT FALSE;
