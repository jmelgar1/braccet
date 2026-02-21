DROP INDEX IF EXISTS idx_matches_forfeit_winner;
ALTER TABLE matches DROP COLUMN IF EXISTS forfeit_winner_id;
