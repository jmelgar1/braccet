-- Link tournaments to an ELO system for automatic rating updates
-- Note: elo_system_id references Community Service, validated via API call (no FK)

ALTER TABLE tournaments ADD COLUMN elo_system_id BIGINT;

CREATE INDEX idx_tournaments_elo_system ON tournaments(elo_system_id) WHERE elo_system_id IS NOT NULL;
