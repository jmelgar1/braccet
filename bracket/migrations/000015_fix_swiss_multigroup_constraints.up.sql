-- Fix swiss tables to support multiple groups per tournament/stage
-- The original schema used tournament_id as the primary key for swiss_config,
-- which prevents having multiple Swiss groups in a multi-stage tournament.

-- First, set default values for NULL stage_id and group_id columns
UPDATE swiss_config SET stage_id = 0 WHERE stage_id IS NULL;
UPDATE swiss_config SET group_id = 0 WHERE group_id IS NULL;
UPDATE swiss_standings SET stage_id = 0 WHERE stage_id IS NULL;
UPDATE swiss_standings SET group_id = 0 WHERE group_id IS NULL;
UPDATE swiss_pairing_history SET stage_id = 0 WHERE stage_id IS NULL;
UPDATE swiss_pairing_history SET group_id = 0 WHERE group_id IS NULL;

-- Set default values for future inserts
ALTER TABLE swiss_config ALTER COLUMN stage_id SET DEFAULT 0;
ALTER TABLE swiss_config ALTER COLUMN group_id SET DEFAULT 0;
ALTER TABLE swiss_standings ALTER COLUMN stage_id SET DEFAULT 0;
ALTER TABLE swiss_standings ALTER COLUMN group_id SET DEFAULT 0;
ALTER TABLE swiss_pairing_history ALTER COLUMN stage_id SET DEFAULT 0;
ALTER TABLE swiss_pairing_history ALTER COLUMN group_id SET DEFAULT 0;

-- 1. Fix swiss_config: change primary key from (tournament_id) to (tournament_id, stage_id, group_id)
ALTER TABLE swiss_config DROP CONSTRAINT swiss_config_pkey;
ALTER TABLE swiss_config ADD CONSTRAINT swiss_config_pkey PRIMARY KEY (tournament_id, stage_id, group_id);

-- 2. Fix swiss_standings: update unique constraint to include stage_id and group_id
ALTER TABLE swiss_standings DROP CONSTRAINT swiss_standings_tournament_id_participant_id_key;
ALTER TABLE swiss_standings ADD CONSTRAINT swiss_standings_unique_participant UNIQUE (tournament_id, stage_id, group_id, participant_id);

-- 3. Fix swiss_pairing_history: update unique constraint to include stage_id and group_id
ALTER TABLE swiss_pairing_history DROP CONSTRAINT swiss_pairing_history_tournament_id_participant1_id_partici_key;
ALTER TABLE swiss_pairing_history ADD CONSTRAINT swiss_pairing_history_unique_pairing UNIQUE (tournament_id, stage_id, group_id, participant1_id, participant2_id);

-- Add comments explaining the changes
COMMENT ON CONSTRAINT swiss_config_pkey ON swiss_config IS 'Composite PK to support multiple Swiss groups per tournament';
COMMENT ON CONSTRAINT swiss_standings_unique_participant ON swiss_standings IS 'Unique participant per tournament/stage/group';
COMMENT ON CONSTRAINT swiss_pairing_history_unique_pairing ON swiss_pairing_history IS 'Unique pairing per tournament/stage/group';
