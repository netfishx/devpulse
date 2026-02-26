DROP INDEX IF EXISTS idx_activities_dedup;
ALTER TABLE activities DROP COLUMN IF EXISTS external_id;
