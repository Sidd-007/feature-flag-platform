-- Add published field to flags table
ALTER TABLE flags ADD COLUMN published BOOLEAN NOT NULL DEFAULT false;

-- Add index for performance
CREATE INDEX idx_flags_published ON flags(env_id, published);
