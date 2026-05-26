-- Seed sync_config for existing master instances that have no rows yet.
-- Previously sync_config was seeded for slave instances; now it belongs to the master.
-- Delete any existing slave sync_config rows, then insert all config types for the master.
DELETE FROM sync_config
WHERE instance_id IN (SELECT id FROM instances WHERE is_master = 0);

INSERT INTO sync_config(instance_id, config_type, enabled)
SELECT i.id, ct.config_type, 1
FROM instances i
CROSS JOIN (
    SELECT 'blocked_services' AS config_type UNION ALL
    SELECT 'clients'          UNION ALL
    SELECT 'dhcp'             UNION ALL
    SELECT 'dns'              UNION ALL
    SELECT 'filtering'        UNION ALL
    SELECT 'parental'         UNION ALL
    SELECT 'rewrite'          UNION ALL
    SELECT 'safebrowsing'     UNION ALL
    SELECT 'safesearch'       UNION ALL
    SELECT 'tls'
) ct
WHERE i.is_master = 1
  AND NOT EXISTS (
      SELECT 1 FROM sync_config sc
      WHERE sc.instance_id = i.id AND sc.config_type = ct.config_type
  );
