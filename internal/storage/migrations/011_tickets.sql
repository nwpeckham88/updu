-- 011_tickets.sql
-- Drop pruned tickets table and seed wan zone
DROP TABLE IF EXISTS tickets;
INSERT OR IGNORE INTO zones (id, name, description) VALUES 
    ('wan', 'Public WAN / External', 'External SaaS and public internet targets');
