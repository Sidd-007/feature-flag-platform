-- Add more flag types to the existing check constraint
ALTER TABLE flags DROP CONSTRAINT IF EXISTS flags_type_check;
ALTER TABLE flags ADD CONSTRAINT flags_type_check CHECK (type IN ('boolean', 'string', 'number', 'json', 'multivariate'));
