-- Add Swiss threshold configuration to tournament_stages
-- These are passed to the bracket service when creating Swiss group brackets
ALTER TABLE tournament_stages
ADD COLUMN wins_to_advance INT,
ADD COLUMN losses_to_eliminate INT;

COMMENT ON COLUMN tournament_stages.wins_to_advance IS 'For Swiss stages: number of wins needed to advance to next stage';
COMMENT ON COLUMN tournament_stages.losses_to_eliminate IS 'For Swiss stages: number of losses that eliminates a participant';
