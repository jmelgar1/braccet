-- Reverse the pending_registrations table creation

DROP INDEX IF EXISTS idx_pending_registrations_expires_at;
DROP INDEX IF EXISTS idx_pending_registrations_token;
DROP TABLE IF EXISTS pending_registrations;
