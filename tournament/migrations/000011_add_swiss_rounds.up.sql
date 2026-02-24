-- Add swiss_rounds column to tournaments table
-- This allows users to override the default number of rounds for Swiss format
ALTER TABLE tournaments ADD COLUMN swiss_rounds INT;
