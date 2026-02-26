-- Add visual position fields for DAG editor display
ALTER TABLE event_tournaments
ADD COLUMN position_x INT NOT NULL DEFAULT 0,
ADD COLUMN position_y INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN event_tournaments.position_x IS 'X coordinate in visual DAG editor';
COMMENT ON COLUMN event_tournaments.position_y IS 'Y coordinate in visual DAG editor';
