-- Create segments table for user targeting
CREATE TABLE IF NOT EXISTS segments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    env_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    key VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rules_json JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version INTEGER NOT NULL DEFAULT 1,

    -- Ensure unique keys within an environment
    UNIQUE(env_id, key)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_segments_env_id ON segments(env_id);
CREATE INDEX IF NOT EXISTS idx_segments_key ON segments(env_id, key);
CREATE INDEX IF NOT EXISTS idx_segments_active ON segments(env_id, is_active);
CREATE INDEX IF NOT EXISTS idx_segments_created_at ON segments(created_at DESC);

-- Create a function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_segments_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update updated_at
CREATE TRIGGER trigger_segments_updated_at
    BEFORE UPDATE ON segments
    FOR EACH ROW
    EXECUTE FUNCTION update_segments_updated_at();

-- Add some sample segments for testing
INSERT INTO segments (env_id, key, name, description, rules_json) 
SELECT 
    e.id,
    'beta-users',
    'Beta Users',
    'Users who opted into beta features',
    '{"conditions": [{"attribute": "beta_user", "operator": "equals", "value": true}]}'
FROM environments e 
WHERE e.key = 'development'
ON CONFLICT (env_id, key) DO NOTHING;

INSERT INTO segments (env_id, key, name, description, rules_json) 
SELECT 
    e.id,
    'premium-users',
    'Premium Users',
    'Users with premium subscription',
    '{"conditions": [{"attribute": "plan", "operator": "equals", "value": "premium"}]}'
FROM environments e 
WHERE e.key = 'development'
ON CONFLICT (env_id, key) DO NOTHING;
