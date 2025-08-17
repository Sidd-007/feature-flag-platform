-- Revert to original flag types
ALTER TABLE flags DROP CONSTRAINT IF EXISTS flags_type_check;
ALTER TABLE flags ADD CONSTRAINT flags_type_check CHECK (type IN ('boolean', 'multivariate', 'json'));
