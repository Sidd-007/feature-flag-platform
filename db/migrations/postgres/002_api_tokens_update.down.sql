-- Rollback API tokens table updates

-- Remove added columns
ALTER TABLE api_tokens 
DROP COLUMN IF EXISTS description,
DROP COLUMN IF EXISTS prefix,
DROP COLUMN IF EXISTS is_active;

-- Rename back
ALTER TABLE api_tokens RENAME COLUMN hashed_token TO hashed_secret;

-- Restore original constraint
ALTER TABLE api_tokens DROP CONSTRAINT IF EXISTS api_tokens_scope_check;
ALTER TABLE api_tokens ADD CONSTRAINT api_tokens_check 
    CHECK ((env_id IS NOT NULL AND org_id IS NULL) OR (env_id IS NULL AND org_id IS NOT NULL));

-- Drop indexes
DROP INDEX IF EXISTS idx_api_tokens_is_active;
DROP INDEX IF EXISTS idx_api_tokens_prefix;
DROP INDEX IF EXISTS idx_api_tokens_hashed_token;
