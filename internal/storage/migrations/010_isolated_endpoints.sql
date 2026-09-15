-- 10. Add is_isolated column to service_endpoints table
ALTER TABLE service_endpoints ADD COLUMN is_isolated BOOLEAN NOT NULL DEFAULT 0;
