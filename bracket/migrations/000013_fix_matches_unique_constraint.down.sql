-- Revert to the old unique constraint (without stage_id and group_id)
-- WARNING: This may fail if there are duplicate (tournament_id, bracket_type, round, position) rows

DROP INDEX IF EXISTS matches_tournament_stage_group_position_key;

ALTER TABLE matches ADD CONSTRAINT matches_tournament_id_bracket_type_round_position_key
    UNIQUE (tournament_id, bracket_type, round, "position");
