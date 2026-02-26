ALTER TABLE activities ADD COLUMN external_id text;
CREATE UNIQUE INDEX idx_activities_dedup ON activities (user_id, source, external_id);
