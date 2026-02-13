CREATE TABLE infrastructure_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES applications(id),
    plan_type VARCHAR(20) NOT NULL,
    from_provider VARCHAR(10),
    to_provider VARCHAR(10),
    content TEXT NOT NULL,
    resources JSONB NOT NULL DEFAULT '[]',
    estimated_cost JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plans_app_id ON infrastructure_plans(application_id);
CREATE INDEX idx_plans_type ON infrastructure_plans(plan_type);
