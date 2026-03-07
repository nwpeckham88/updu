-- 005_heartbeat_token.sql
-- Add token column to heartbeats table for security
ALTER TABLE heartbeats ADD COLUMN token TEXT;
