CREATE TABLE IF NOT EXISTS notification_channels (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL UNIQUE,
    type           TEXT NOT NULL,
    config_enc     TEXT NOT NULL,
    notify_success INTEGER NOT NULL DEFAULT 1,
    notify_failure INTEGER NOT NULL DEFAULT 1,
    enabled        INTEGER NOT NULL DEFAULT 1,
    created_at     DATETIME NOT NULL,
    updated_at     DATETIME NOT NULL
);
