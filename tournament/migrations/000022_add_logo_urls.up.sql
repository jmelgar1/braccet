-- Add logo_url column to tournaments table
ALTER TABLE tournaments ADD COLUMN logo_url TEXT;

-- Add logo_url column to events table
ALTER TABLE events ADD COLUMN logo_url TEXT;
