-- Drop Feature Flag Platform initial schema

-- Drop triggers first
DROP TRIGGER IF EXISTS update_user_org_memberships_updated_at ON user_org_memberships;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_api_tokens_updated_at ON api_tokens;
DROP TRIGGER IF EXISTS update_metrics_updated_at ON metrics;
DROP TRIGGER IF EXISTS update_experiments_updated_at ON experiments;
DROP TRIGGER IF EXISTS update_flags_updated_at ON flags;
DROP TRIGGER IF EXISTS update_segments_updated_at ON segments;
DROP TRIGGER IF EXISTS update_environments_updated_at ON environments;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_orgs_updated_at ON orgs;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS user_org_memberships;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS api_tokens;
DROP TABLE IF EXISTS metrics;
DROP TABLE IF EXISTS experiments;
DROP TABLE IF EXISTS flags;
DROP TABLE IF EXISTS segments;
DROP TABLE IF EXISTS environments;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS orgs;
