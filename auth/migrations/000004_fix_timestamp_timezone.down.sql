-- Revert to TIMESTAMP (without timezone)

ALTER TABLE pending_registrations
    ALTER COLUMN expires_at TYPE TIMESTAMP,
    ALTER COLUMN created_at TYPE TIMESTAMP;
