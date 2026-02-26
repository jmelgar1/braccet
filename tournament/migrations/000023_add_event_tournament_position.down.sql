-- Remove visual position fields
ALTER TABLE event_tournaments
DROP COLUMN position_x,
DROP COLUMN position_y;
