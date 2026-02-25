ALTER TABLE tournament_stages
DROP COLUMN IF EXISTS losses_to_eliminate,
DROP COLUMN IF EXISTS wins_to_advance;
