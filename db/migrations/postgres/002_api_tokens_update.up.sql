-- Update API tokens table to match repository structure

-- Add missing columns
ALTER TABLE api_tokens 
ADD COLUMN IF NOT EXISTS description TEXT,
ADD COLUMN IF NOT EXISTS prefix VARCHAR(8),
ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

-- Rename hashed_secret to hashed_token for consistency
ALTER TABLE api_tokens RENAME COLUMN hashed_secret TO hashed_token;

-- Update check constraint to allow environment-scoped tokens only for now
ALTER TABLE api_tokens DROP CONSTRAINT IF EXISTS api_tokens_check;
ALTER TABLE api_tokens ADD CONSTRAINT api_tokens_scope_check 
    CHECK (env_id IS NOT NULL);

-- Create additional indexes
CREATE INDEX IF NOT EXISTS idx_api_tokens_is_active ON api_tokens(is_active);
CREATE INDEX IF NOT EXISTS idx_api_tokens_prefix ON api_tokens(prefix);
CREATE INDEX IF NOT EXISTS idx_api_tokens_hashed_token ON api_tokens(hashed_token) WHERE is_active = true;
