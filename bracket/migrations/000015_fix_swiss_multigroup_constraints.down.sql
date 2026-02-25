-- Revert swiss table constraints back to original schema
-- WARNING: This will fail if there are multiple groups per tournament

-- 1. Revert swiss_config primary key
ALTER TABLE swiss_config DROP CONSTRAINT swiss_config_pkey;
ALTER TABLE swiss_config ADD PRIMARY KEY (tournament_id);

-- 2. Revert swiss_standings unique constraint
ALTER TABLE swiss_standings DROP CONSTRAINT swiss_standings_unique_participant;
ALTER TABLE swiss_standings ADD CONSTRAINT swiss_standings_tournament_id_participant_id_key UNIQUE (tournament_id, participant_id);

-- 3. Revert swiss_pairing_history unique constraint
ALTER TABLE swiss_pairing_history DROP CONSTRAINT swiss_pairing_history_unique_pairing;
ALTER TABLE swiss_pairing_history ADD CONSTRAINT swiss_pairing_history_tournament_id_participant1_id_partici_key UNIQUE (tournament_id, participant1_id, participant2_id);

-- Remove default values
ALTER TABLE swiss_config ALTER COLUMN stage_id DROP DEFAULT;
ALTER TABLE swiss_config ALTER COLUMN group_id DROP DEFAULT;
ALTER TABLE swiss_standings ALTER COLUMN stage_id DROP DEFAULT;
ALTER TABLE swiss_standings ALTER COLUMN group_id DROP DEFAULT;
ALTER TABLE swiss_pairing_history ALTER COLUMN stage_id DROP DEFAULT;
ALTER TABLE swiss_pairing_history ALTER COLUMN group_id DROP DEFAULT;

-- Set values back to NULL for legacy single-tournament records
UPDATE swiss_config SET stage_id = NULL WHERE stage_id = 0;
UPDATE swiss_config SET group_id = NULL WHERE group_id = 0;
UPDATE swiss_standings SET stage_id = NULL WHERE stage_id = 0;
UPDATE swiss_standings SET group_id = NULL WHERE group_id = 0;
UPDATE swiss_pairing_history SET stage_id = NULL WHERE stage_id = 0;
UPDATE swiss_pairing_history SET group_id = NULL WHERE group_id = 0;
