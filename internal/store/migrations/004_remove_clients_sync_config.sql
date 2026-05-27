-- Clients are instance-specific and must not be synced between instances.
DELETE FROM sync_config WHERE config_type = 'clients';
