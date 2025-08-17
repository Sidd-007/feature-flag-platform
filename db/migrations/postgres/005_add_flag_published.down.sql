-- Remove published field from flags table
DROP INDEX IF EXISTS idx_flags_published;
ALTER TABLE flags DROP COLUMN IF EXISTS published;
