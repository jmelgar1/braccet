-- Add 'withdrawn' to participant_status enum
-- Used when a participant voluntarily leaves an in-progress tournament

ALTER TYPE participant_status ADD VALUE 'withdrawn';
