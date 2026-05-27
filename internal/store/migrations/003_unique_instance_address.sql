-- Prevent duplicate instances pointing at the same AdGuardHome server.
CREATE UNIQUE INDEX IF NOT EXISTS idx_instances_address ON instances(address);
