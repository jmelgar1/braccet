-- Add slug column for URL-friendly tournament identifiers
ALTER TABLE tournaments ADD COLUMN slug VARCHAR(8) UNIQUE;

-- Create index for fast slug lookups
CREATE INDEX idx_tournaments_slug ON tournaments(slug);

-- Generate slugs for existing tournaments
DO $$
DECLARE
    t RECORD;
    new_slug VARCHAR(8);
    chars VARCHAR(36) := 'abcdefghijklmnopqrstuvwxyz0123456789';
BEGIN
    FOR t IN SELECT id FROM tournaments WHERE slug IS NULL LOOP
        LOOP
            new_slug := '';
            FOR i IN 1..8 LOOP
                new_slug := new_slug || substr(chars, floor(random() * 36 + 1)::int, 1);
            END LOOP;
            BEGIN
                UPDATE tournaments SET slug = new_slug WHERE id = t.id;
                EXIT;
            EXCEPTION WHEN unique_violation THEN
                -- Retry with new slug
            END;
        END LOOP;
    END LOOP;
END $$;

-- Make slug NOT NULL after populating existing rows
ALTER TABLE tournaments ALTER COLUMN slug SET NOT NULL;
