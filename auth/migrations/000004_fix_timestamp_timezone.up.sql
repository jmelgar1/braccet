-- Fix timezone handling: change TIMESTAMP to TIMESTAMPTZ
-- This ensures proper timezone conversion between Go and PostgreSQL

ALTER TABLE pending_registrations
    ALTER COLUMN expires_at TYPE TIMESTAMPTZ,
    ALTER COLUMN created_at TYPE TIMESTAMPTZ;
