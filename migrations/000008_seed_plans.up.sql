-- Seed Initial Plans
-- Default subscription tiers

INSERT INTO plans (id, type, name, description, billing_cycle, price_usd, monthly_tokens, daily_requests, max_api_keys, features, display_order) VALUES
('free', 'free', 'Free', 'Free tier for getting started', NULL, 0, 10000, 100, 2,
 ARRAY['10K tokens/month', '100 requests/day', 'Basic model access', '2 API keys'], 1),

('pro_monthly', 'pro', 'Pro Monthly', 'Professional plan - monthly billing', 'monthly', 2000, 500000, 2000, 5,
 ARRAY['500K tokens/month', '2000 requests/day', 'All model access', 'Priority support', '5 API keys'], 2),

('pro_yearly', 'pro', 'Pro Yearly', 'Professional plan - annual billing (2 months free)', 'yearly', 20000, 500000, 2000, 5,
 ARRAY['500K tokens/month', '2000 requests/day', 'All model access', 'Priority support', '5 API keys', '2 months free'], 3),

('team_monthly', 'team', 'Team Monthly', 'Team plan - monthly billing', 'monthly', 5000, 2000000, 10000, 20,
 ARRAY['2M tokens/month', '10K requests/day', 'Team collaboration', 'API priority', 'Dedicated support', '20 API keys'], 4),

('team_yearly', 'team', 'Team Yearly', 'Team plan - annual billing (2 months free)', 'yearly', 50000, 2000000, 10000, 20,
 ARRAY['2M tokens/month', '10K requests/day', 'Team collaboration', 'API priority', 'Dedicated support', '20 API keys', '2 months free'], 5),

('enterprise', 'enterprise', 'Enterprise', 'Enterprise plan - custom pricing', NULL, -1, -1, -1, -1,
 ARRAY['Unlimited tokens', 'Unlimited requests', 'Custom deployment', 'SLA guarantee', 'Dedicated account manager', 'Custom integrations'], 6)

ON CONFLICT (id) DO UPDATE SET
    type = EXCLUDED.type,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    billing_cycle = EXCLUDED.billing_cycle,
    price_usd = EXCLUDED.price_usd,
    monthly_tokens = EXCLUDED.monthly_tokens,
    daily_requests = EXCLUDED.daily_requests,
    max_api_keys = EXCLUDED.max_api_keys,
    features = EXCLUDED.features,
    display_order = EXCLUDED.display_order,
    updated_at = CURRENT_TIMESTAMP;
