-- Seed default AI providers and models
-- NOTE: Replace API keys with actual values before running

-- OpenAI Provider
INSERT INTO ai_providers (id, name, type, base_url, api_key, enabled, weight, priority, options)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'OpenAI',
    'openai',
    'https://api.openai.com/v1',
    'sk-YOUR-OPENAI-API-KEY',
    true,
    100,
    10,
    '{"organization": ""}'
);

-- OpenAI Models
INSERT INTO ai_models (id, provider_id, name, capabilities, context_window, max_output_tokens, input_cost_per_1k, output_cost_per_1k, enabled, options) VALUES
('gpt-4o', 'a0000000-0000-0000-0000-000000000001', 'GPT-4o', ARRAY['chat', 'vision', 'function_calling'], 128000, 16384, 0.005, 0.015, true, '{}'),
('gpt-4o-mini', 'a0000000-0000-0000-0000-000000000001', 'GPT-4o Mini', ARRAY['chat', 'vision', 'function_calling'], 128000, 16384, 0.00015, 0.0006, true, '{}'),
('gpt-4-turbo', 'a0000000-0000-0000-0000-000000000001', 'GPT-4 Turbo', ARRAY['chat', 'vision', 'function_calling'], 128000, 4096, 0.01, 0.03, true, '{}'),
('gpt-3.5-turbo', 'a0000000-0000-0000-0000-000000000001', 'GPT-3.5 Turbo', ARRAY['chat', 'function_calling'], 16385, 4096, 0.0005, 0.0015, true, '{}'),
('text-embedding-3-large', 'a0000000-0000-0000-0000-000000000001', 'Text Embedding 3 Large', ARRAY['embedding'], 8191, 0, 0.00013, 0, true, '{"dimensions": 3072}'),
('text-embedding-3-small', 'a0000000-0000-0000-0000-000000000001', 'Text Embedding 3 Small', ARRAY['embedding'], 8191, 0, 0.00002, 0, true, '{"dimensions": 1536}'),
('dall-e-3', 'a0000000-0000-0000-0000-000000000001', 'DALL-E 3', ARRAY['image_generation'], 0, 0, 0.04, 0, true, '{"sizes": ["1024x1024", "1792x1024", "1024x1792"]}');

-- Anthropic Provider
INSERT INTO ai_providers (id, name, type, base_url, api_key, enabled, weight, priority, options)
VALUES (
    'a0000000-0000-0000-0000-000000000002',
    'Anthropic',
    'anthropic',
    'https://api.anthropic.com',
    'sk-ant-YOUR-ANTHROPIC-API-KEY',
    true,
    100,
    10,
    '{}'
);

-- Anthropic Models
INSERT INTO ai_models (id, provider_id, name, capabilities, context_window, max_output_tokens, input_cost_per_1k, output_cost_per_1k, enabled, options) VALUES
('claude-3-5-sonnet-20241022', 'a0000000-0000-0000-0000-000000000002', 'Claude 3.5 Sonnet', ARRAY['chat', 'vision'], 200000, 8192, 0.003, 0.015, true, '{}'),
('claude-3-5-haiku-20241022', 'a0000000-0000-0000-0000-000000000002', 'Claude 3.5 Haiku', ARRAY['chat', 'vision'], 200000, 8192, 0.001, 0.005, true, '{}'),
('claude-3-opus-20240229', 'a0000000-0000-0000-0000-000000000002', 'Claude 3 Opus', ARRAY['chat', 'vision'], 200000, 4096, 0.015, 0.075, true, '{}');

-- Default Model Groups
INSERT INTO ai_groups (id, name, task_type, models, strategy, fallback, required_capabilities, enabled) VALUES
(
    'default-chat',
    'Default Chat',
    'chat',
    ARRAY['gpt-4o-mini', 'claude-3-5-haiku-20241022', 'gpt-3.5-turbo'],
    '{"type": "cost_optimized", "options": {"max_cost_per_1k": 0.01}}',
    '{"models": ["gpt-3.5-turbo"], "strategy": {"type": "round_robin"}}',
    ARRAY['chat'],
    true
),
(
    'premium-chat',
    'Premium Chat',
    'chat',
    ARRAY['gpt-4o', 'claude-3-5-sonnet-20241022', 'claude-3-opus-20240229'],
    '{"type": "priority", "options": {}}',
    '{"models": ["gpt-4o-mini"], "strategy": {"type": "round_robin"}}',
    ARRAY['chat'],
    true
),
(
    'vision-chat',
    'Vision Chat',
    'chat',
    ARRAY['gpt-4o', 'gpt-4o-mini', 'claude-3-5-sonnet-20241022'],
    '{"type": "cost_optimized", "options": {}}',
    NULL,
    ARRAY['chat', 'vision'],
    true
),
(
    'default-embedding',
    'Default Embedding',
    'embedding',
    ARRAY['text-embedding-3-small', 'text-embedding-3-large'],
    '{"type": "priority", "options": {}}',
    NULL,
    ARRAY['embedding'],
    true
),
(
    'default-image',
    'Default Image Generation',
    'image_generation',
    ARRAY['dall-e-3'],
    '{"type": "priority", "options": {}}',
    NULL,
    ARRAY['image_generation'],
    true
);
