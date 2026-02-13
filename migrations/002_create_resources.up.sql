CREATE TABLE resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    kind VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    spec JSONB NOT NULL DEFAULT '{}',
    provider_mappings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_resources_app_id ON resources(application_id);
