-- Store pending registrations awaiting email verification
-- Users are only created in the users table after email verification

CREATE TABLE pending_registrations (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(50) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,

    -- Verification token (32 bytes = 256 bits of entropy, hex-encoded = 64 chars)
    verification_token VARCHAR(64) NOT NULL UNIQUE,

    -- Expiration and tracking
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for token lookups (primary verification path)
CREATE INDEX idx_pending_registrations_token ON pending_registrations(verification_token);

-- Index for cleanup of expired registrations
CREATE INDEX idx_pending_registrations_expires_at ON pending_registrations(expires_at);
