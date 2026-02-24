-- Note: PostgreSQL does not support removing enum values directly.
-- To rollback, you would need to:
-- 1. Create a new enum type without 'swiss'
-- 2. Update columns to use the new type
-- 3. Drop the old type and rename the new one
-- This is left as a manual operation since it requires data migration.

-- For development, you can recreate the database or manually handle this.
