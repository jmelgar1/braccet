-- Restore score columns to matches table
ALTER TABLE matches ADD COLUMN participant1_score INT;
ALTER TABLE matches ADD COLUMN participant2_score INT;

-- Drop match_sets table
DROP TABLE IF EXISTS match_sets;
