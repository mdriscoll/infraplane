ALTER TABLE deployments ADD COLUMN plan_id UUID REFERENCES infrastructure_plans(id);
