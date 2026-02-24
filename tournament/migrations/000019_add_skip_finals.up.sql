-- Add skip_finals column to tournament_stages
-- When true, final matches are skipped and standings determined by bracket position only
ALTER TABLE tournament_stages ADD COLUMN skip_finals BOOLEAN DEFAULT FALSE;
