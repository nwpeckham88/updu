-- 004_multiple_groups.sql
-- Add groups JSON column and migrate data from group_name

ALTER TABLE monitors ADD COLUMN groups JSON DEFAULT '[]';

-- Migrate existing data
UPDATE monitors SET groups = json_array(group_name) WHERE group_name IS NOT NULL AND group_name != '';

-- Note: We are NOT dropping group_name yet to maintain backward compatibility during migration 
-- and because some SQLite versions have limited ALTER TABLE support.
-- We will stop using group_name in queries.
