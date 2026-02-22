-- Revert to non-cascading FK constraints on deployments and infrastructure_plans.

ALTER TABLE deployments
    DROP CONSTRAINT deployments_application_id_fkey,
    ADD CONSTRAINT deployments_application_id_fkey
        FOREIGN KEY (application_id) REFERENCES applications(id);

ALTER TABLE infrastructure_plans
    DROP CONSTRAINT infrastructure_plans_application_id_fkey,
    ADD CONSTRAINT infrastructure_plans_application_id_fkey
        FOREIGN KEY (application_id) REFERENCES applications(id);
