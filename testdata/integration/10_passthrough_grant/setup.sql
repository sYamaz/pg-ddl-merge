-- Roles referenced by GRANT statements in input files.
-- Applied before both sequential and merged runs.
DO $$ BEGIN CREATE ROLE readonly_role; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE ROLE app_role; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE ROLE reporting_role; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
