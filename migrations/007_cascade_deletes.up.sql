-- Fix missing ON DELETE CASCADE on deployments and infrastructure_plans.
-- Without this, deleting an application fails if it has deployments or plans.

ALTER TABLE deployments
    DROP CONSTRAINT deployments_application_id_fkey,
    ADD CONSTRAINT deployments_application_id_fkey
        FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;

ALTER TABLE infrastructure_plans
    DROP CONSTRAINT infrastructure_plans_application_id_fkey,
    ADD CONSTRAINT infrastructure_plans_application_id_fkey
        FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;
