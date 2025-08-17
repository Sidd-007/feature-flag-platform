-- Initialize Feature Flags database
CREATE DATABASE feature_flags;

-- Create extensions
\c feature_flags;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE feature_flags TO postgres;
