-- Remove logo_url column from tournaments table
ALTER TABLE tournaments DROP COLUMN IF EXISTS logo_url;

-- Remove logo_url column from events table
ALTER TABLE events DROP COLUMN IF EXISTS logo_url;
