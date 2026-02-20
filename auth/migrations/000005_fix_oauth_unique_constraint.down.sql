-- Revert to the NULLS NOT DISTINCT constraint

DROP INDEX users_oauth_unique;

ALTER TABLE users ADD CONSTRAINT users_oauth_unique
    UNIQUE NULLS NOT DISTINCT (oauth_provider, oauth_id);
