-- Rollback: Seed Initial Plans

DELETE FROM plans WHERE id IN (
    'free',
    'pro_monthly',
    'pro_yearly',
    'team_monthly',
    'team_yearly',
    'enterprise'
);
