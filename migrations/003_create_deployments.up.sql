CREATE TABLE deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES applications(id),
    provider VARCHAR(10) NOT NULL,
    git_commit VARCHAR(40) NOT NULL DEFAULT '',
    git_branch VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    terraform_plan TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_deployments_app_id ON deployments(application_id);
CREATE INDEX idx_deployments_status ON deployments(status);
