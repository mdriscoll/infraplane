CREATE TABLE analysis_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    trigger VARCHAR(50) NOT NULL,
    resource_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_analysis_runs_app_id ON analysis_runs(application_id);

ALTER TABLE resources ADD COLUMN analysis_run_id UUID REFERENCES analysis_runs(id) ON DELETE CASCADE;
CREATE INDEX idx_resources_run_id ON resources(analysis_run_id);
