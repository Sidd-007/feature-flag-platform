-- Create initial schema for Feature Flag Platform

-- Organizations table
CREATE TABLE IF NOT EXISTS orgs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    billing_tier VARCHAR(50) NOT NULL DEFAULT 'free',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1
);

-- Projects table
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1,
    UNIQUE(org_id, key)
);

-- Environments table
CREATE TABLE IF NOT EXISTS environments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key VARCHAR(100) NOT NULL,
    salt VARCHAR(255) NOT NULL DEFAULT encode(gen_random_bytes(32), 'hex'),
    is_prod BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1,
    UNIQUE(project_id, key)
);

-- Segments table (for user targeting)
CREATE TABLE IF NOT EXISTS segments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    env_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key VARCHAR(100) NOT NULL,
    description TEXT,
    rules_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1,
    UNIQUE(env_id, key)
);

-- Feature flags table
CREATE TABLE IF NOT EXISTS flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    env_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    key VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL CHECK (type IN ('boolean', 'multivariate', 'json')),
    variations JSONB NOT NULL DEFAULT '[]',
    default_variation VARCHAR(100) NOT NULL,
    rules_json JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1,
    UNIQUE(env_id, key)
);

-- Experiments table
CREATE TABLE IF NOT EXISTS experiments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    env_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    flag_id UUID NOT NULL REFERENCES flags(id) ON DELETE CASCADE,
    key VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    hypothesis TEXT,
    variations_map JSONB NOT NULL DEFAULT '{}',
    primary_metric_id UUID,
    secondary_metric_ids UUID[] DEFAULT '{}',
    start_at TIMESTAMP WITH TIME ZONE,
    stop_at TIMESTAMP WITH TIME ZONE,
    traffic_allocation DECIMAL(5,4) DEFAULT 1.0 CHECK (traffic_allocation >= 0 AND traffic_allocation <= 1),
    status VARCHAR(50) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'running', 'stopped', 'completed')),
    exclusion_group VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1,
    UNIQUE(env_id, key)
);

-- Metrics table (for experiment tracking)
CREATE TABLE IF NOT EXISTS metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    env_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    key VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL CHECK (type IN ('binary', 'ratio', 'continuous')),
    unit VARCHAR(50),
    higher_is_better BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1,
    UNIQUE(env_id, key)
);

-- API tokens table
CREATE TABLE IF NOT EXISTS api_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    env_id UUID REFERENCES environments(id) ON DELETE CASCADE,
    org_id UUID REFERENCES orgs(id) ON DELETE CASCADE,
    scope VARCHAR(100) NOT NULL CHECK (scope IN ('read', 'write', 'admin')),
    hashed_secret TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1,
    CHECK ((env_id IS NOT NULL AND org_id IS NULL) OR (env_id IS NULL AND org_id IS NOT NULL))
);

-- Users table (for local authentication)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    email_verified BOOLEAN DEFAULT FALSE,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1
);

-- User organization memberships table
CREATE TABLE IF NOT EXISTS user_org_memberships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('owner', 'admin', 'editor', 'viewer')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, org_id)
);

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor_id UUID REFERENCES users(id),
    actor_type VARCHAR(50) NOT NULL CHECK (actor_type IN ('user', 'api_token', 'system')),
    actor_name VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID,
    resource_name VARCHAR(255),
    org_id UUID REFERENCES orgs(id),
    project_id UUID REFERENCES projects(id),
    env_id UUID REFERENCES environments(id),
    diff_json JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add foreign key constraint for metrics in experiments
ALTER TABLE experiments ADD CONSTRAINT fk_experiments_primary_metric 
    FOREIGN KEY (primary_metric_id) REFERENCES metrics(id);

-- Create indexes for performance
CREATE INDEX idx_orgs_slug ON orgs(slug);
CREATE INDEX idx_projects_org_id ON projects(org_id);
CREATE INDEX idx_projects_key ON projects(org_id, key);
CREATE INDEX idx_environments_project_id ON environments(project_id);
CREATE INDEX idx_environments_key ON environments(project_id, key);
CREATE INDEX idx_segments_env_id ON segments(env_id);
CREATE INDEX idx_segments_key ON segments(env_id, key);
CREATE INDEX idx_flags_env_id ON flags(env_id);
CREATE INDEX idx_flags_key ON flags(env_id, key);
CREATE INDEX idx_flags_status ON flags(env_id, status);
CREATE INDEX idx_experiments_env_id ON experiments(env_id);
CREATE INDEX idx_experiments_flag_id ON experiments(flag_id);
CREATE INDEX idx_experiments_status ON experiments(env_id, status);
CREATE INDEX idx_experiments_exclusion_group ON experiments(env_id, exclusion_group) WHERE exclusion_group IS NOT NULL;
CREATE INDEX idx_metrics_env_id ON metrics(env_id);
CREATE INDEX idx_metrics_key ON metrics(env_id, key);
CREATE INDEX idx_api_tokens_env_id ON api_tokens(env_id) WHERE env_id IS NOT NULL;
CREATE INDEX idx_api_tokens_org_id ON api_tokens(org_id) WHERE org_id IS NOT NULL;
CREATE INDEX idx_api_tokens_expires_at ON api_tokens(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_user_org_memberships_user_id ON user_org_memberships(user_id);
CREATE INDEX idx_user_org_memberships_org_id ON user_org_memberships(org_id);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs(actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id) WHERE resource_id IS NOT NULL;
CREATE INDEX idx_audit_logs_org_id ON audit_logs(org_id) WHERE org_id IS NOT NULL;
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    NEW.version = OLD.version + 1;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_orgs_updated_at BEFORE UPDATE ON orgs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_environments_updated_at BEFORE UPDATE ON environments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_segments_updated_at BEFORE UPDATE ON segments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_flags_updated_at BEFORE UPDATE ON flags FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_experiments_updated_at BEFORE UPDATE ON experiments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_metrics_updated_at BEFORE UPDATE ON metrics FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_api_tokens_updated_at BEFORE UPDATE ON api_tokens FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_org_memberships_updated_at BEFORE UPDATE ON user_org_memberships FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
