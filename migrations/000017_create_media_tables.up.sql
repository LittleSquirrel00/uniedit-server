-- Media providers table
CREATE TABLE IF NOT EXISTS media_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    base_url VARCHAR(500) NOT NULL,
    encrypted_key TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_media_providers_enabled ON media_providers(enabled);

-- Media models table
CREATE TABLE IF NOT EXISTS media_models (
    id VARCHAR(100) PRIMARY KEY,
    provider_id UUID NOT NULL REFERENCES media_providers(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    capabilities JSONB NOT NULL DEFAULT '[]',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_media_models_provider ON media_models(provider_id);
CREATE INDEX idx_media_models_enabled ON media_models(enabled);

-- Media tasks table
CREATE TABLE IF NOT EXISTS media_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    progress INT NOT NULL DEFAULT 0,
    input JSONB NOT NULL DEFAULT '{}',
    output JSONB,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_media_tasks_owner ON media_tasks(owner_id);
CREATE INDEX idx_media_tasks_status ON media_tasks(status);
CREATE INDEX idx_media_tasks_created ON media_tasks(created_at DESC);

-- Seed default media providers
INSERT INTO media_providers (id, name, type, base_url, enabled) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'OpenAI', 'openai', 'https://api.openai.com/v1', true),
    ('a0000000-0000-0000-0000-000000000002', 'Replicate', 'generic', 'https://api.replicate.com/v1', true)
ON CONFLICT (id) DO NOTHING;

-- Seed default media models
INSERT INTO media_models (id, provider_id, name, capabilities, enabled) VALUES
    ('dall-e-3', 'a0000000-0000-0000-0000-000000000001', 'DALL-E 3', '["image"]', true),
    ('dall-e-2', 'a0000000-0000-0000-0000-000000000001', 'DALL-E 2', '["image"]', true),
    ('runway-gen2', 'a0000000-0000-0000-0000-000000000002', 'Runway Gen-2', '["video"]', true),
    ('stable-video', 'a0000000-0000-0000-0000-000000000002', 'Stable Video Diffusion', '["video"]', true)
ON CONFLICT (id) DO NOTHING;
