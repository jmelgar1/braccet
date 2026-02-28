-- Add expected_participants column to tournament_stages
-- This allows organizers to manually specify expected participant count per stage
-- for accurate prize tier calculation in multi-stage tournaments
ALTER TABLE tournament_stages ADD COLUMN expected_participants INTEGER;
