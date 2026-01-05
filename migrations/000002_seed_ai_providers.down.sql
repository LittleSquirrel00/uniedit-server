-- Remove seed data
DELETE FROM ai_groups WHERE id IN (
    'default-chat',
    'premium-chat',
    'vision-chat',
    'default-embedding',
    'default-image'
);

DELETE FROM ai_models WHERE provider_id IN (
    'a0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000002'
);

DELETE FROM ai_providers WHERE id IN (
    'a0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000002'
);
