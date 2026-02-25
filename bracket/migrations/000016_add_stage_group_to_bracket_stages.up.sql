-- Add stage_id and group_id columns to bracket_stages for multi-stage tournament support
-- Use IF NOT EXISTS to handle cases where columns already exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bracket_stages' AND column_name = 'stage_id') THEN
        ALTER TABLE bracket_stages ADD COLUMN stage_id BIGINT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bracket_stages' AND column_name = 'group_id') THEN
        ALTER TABLE bracket_stages ADD COLUMN group_id BIGINT;
    END IF;
END $$;

-- Drop the old unique constraint
ALTER TABLE bracket_stages DROP CONSTRAINT IF EXISTS bracket_stages_tournament_id_bracket_type_round_key;

-- Create new unique constraint that includes stage_id and group_id
-- Using COALESCE to handle NULL values - NULL values are treated as 0 for uniqueness
DROP INDEX IF EXISTS bracket_stages_unique_idx;
CREATE UNIQUE INDEX bracket_stages_unique_idx ON bracket_stages (
    tournament_id,
    COALESCE(stage_id, 0),
    COALESCE(group_id, 0),
    bracket_type,
    round
);

-- Create index for group-based lookups
DROP INDEX IF EXISTS idx_bracket_stages_group;
CREATE INDEX idx_bracket_stages_group ON bracket_stages(tournament_id, stage_id, group_id);
