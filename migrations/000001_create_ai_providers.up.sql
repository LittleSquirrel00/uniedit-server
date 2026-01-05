-- AI Providers table
CREATE TABLE ai_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    base_url TEXT NOT NULL,
    api_key TEXT NOT NULL,
    enabled BOOLEAN DEFAULT true,
    weight INT DEFAULT 1,
    priority INT DEFAULT 0,
    rate_limit JSONB,
    options JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_providers_type ON ai_providers(type);
CREATE INDEX idx_ai_providers_enabled ON ai_providers(enabled) WHERE enabled = true;

-- AI Models table
CREATE TABLE ai_models (
    id VARCHAR(255) PRIMARY KEY,
    provider_id UUID NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    capabilities TEXT[] NOT NULL,
    context_window INT NOT NULL,
    max_output_tokens INT NOT NULL,
    input_cost_per_1k DECIMAL(10, 6) NOT NULL,
    output_cost_per_1k DECIMAL(10, 6) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    options JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_models_provider ON ai_models(provider_id);
CREATE INDEX idx_ai_models_capabilities ON ai_models USING GIN(capabilities);
CREATE INDEX idx_ai_models_enabled ON ai_models(enabled) WHERE enabled = true;

-- AI Groups table
CREATE TABLE ai_groups (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    task_type VARCHAR(50) NOT NULL,
    models TEXT[] NOT NULL,
    strategy JSONB NOT NULL,
    fallback JSONB,
    required_capabilities TEXT[],
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_groups_task_type ON ai_groups(task_type);
CREATE INDEX idx_ai_groups_enabled ON ai_groups(enabled) WHERE enabled = true;

-- AI Tasks table
CREATE TABLE ai_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    progress INT DEFAULT 0,
    input JSONB NOT NULL,
    output JSONB,
    error JSONB,
    external_task_id VARCHAR(255),
    provider_id UUID REFERENCES ai_providers(id),
    model_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_ai_tasks_user ON ai_tasks(user_id, created_at DESC);
CREATE INDEX idx_ai_tasks_status ON ai_tasks(status) WHERE status IN ('pending', 'running');
CREATE INDEX idx_ai_tasks_external ON ai_tasks(external_task_id) WHERE external_task_id IS NOT NULL;
