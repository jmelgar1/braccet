-- Add threshold mode to swiss_config for advance after X wins / eliminate after X losses
ALTER TABLE swiss_config
ADD COLUMN wins_to_advance INT,
ADD COLUMN losses_to_eliminate INT;

-- Add status tracking to swiss_standings
-- 'active' = still playing
-- 'advanced' = reached wins_to_advance threshold
-- 'eliminated' = reached losses_to_eliminate threshold
CREATE TYPE swiss_participant_status AS ENUM ('active', 'advanced', 'eliminated');

ALTER TABLE swiss_standings
ADD COLUMN status swiss_participant_status DEFAULT 'active',
ADD COLUMN match_wins INT DEFAULT 0;  -- Real wins (excluding BYEs)

-- Index for efficient filtering of active participants
CREATE INDEX idx_swiss_standings_status ON swiss_standings(tournament_id, status);

-- Add comments
COMMENT ON COLUMN swiss_config.wins_to_advance IS 'Number of real match wins (excluding BYEs) needed to advance';
COMMENT ON COLUMN swiss_config.losses_to_eliminate IS 'Number of losses that eliminates a participant';
COMMENT ON COLUMN swiss_standings.status IS 'Participant status: active (still playing), advanced (reached wins threshold), eliminated (reached losses threshold)';
COMMENT ON COLUMN swiss_standings.match_wins IS 'Real match wins excluding BYEs - used for advancement threshold';
