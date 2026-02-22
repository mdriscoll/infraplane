DROP INDEX IF EXISTS idx_resources_run_id;
ALTER TABLE resources DROP COLUMN IF EXISTS analysis_run_id;
DROP TABLE IF EXISTS analysis_runs;
