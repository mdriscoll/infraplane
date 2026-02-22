ALTER TABLE applications ADD COLUMN compliance_frameworks JSONB NOT NULL DEFAULT '[]'::jsonb;
