-- Drop triggers
DROP TRIGGER IF EXISTS trigger_segments_updated_at ON segments;

-- Drop functions
DROP FUNCTION IF EXISTS update_segments_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_segments_created_at;
DROP INDEX IF EXISTS idx_segments_active;
DROP INDEX IF EXISTS idx_segments_key;
DROP INDEX IF EXISTS idx_segments_env_id;

-- Drop segments table
DROP TABLE IF EXISTS segments;
