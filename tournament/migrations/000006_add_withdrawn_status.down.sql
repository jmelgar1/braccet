-- PostgreSQL cannot easily remove enum values
-- This requires recreating the type and migrating data
-- For safety, this migration is not reversible without manual intervention

-- To reverse this migration:
-- 1. Ensure no participants have 'withdrawn' status
-- 2. Create a new type without 'withdrawn'
-- 3. Alter the column to use the new type
-- 4. Drop the old type

-- WARNING: This down migration does nothing automatically
-- Manual intervention required if rollback is needed
