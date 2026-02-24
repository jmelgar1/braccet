-- Fix unique constraint on matches to include stage_id and group_id
-- This allows multiple groups to have matches with the same (bracket_type, round, position)

-- Drop the old constraint that doesn't account for groups
ALTER TABLE matches DROP CONSTRAINT IF EXISTS matches_tournament_id_bracket_type_round_position_key;

-- Create a new unique index that handles NULLs properly for stage_id and group_id
-- Using COALESCE to treat NULL as 0 for uniqueness purposes
CREATE UNIQUE INDEX matches_tournament_stage_group_position_key
ON matches (
    tournament_id,
    COALESCE(stage_id, 0),
    COALESCE(group_id, 0),
    bracket_type,
    round,
    "position"
);
