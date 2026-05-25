CREATE TABLE IF NOT EXISTS instances (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    address         TEXT NOT NULL,
    username        TEXT NOT NULL DEFAULT '',
    password_enc    TEXT NOT NULL DEFAULT '',
    is_master       INTEGER NOT NULL DEFAULT 0,
    tls_skip_verify INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS sync_config (
    instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    config_type TEXT NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (instance_id, config_type)
);

CREATE TABLE IF NOT EXISTS sync_runs (
    id           TEXT PRIMARY KEY,
    triggered_by TEXT NOT NULL,
    started_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    finished_at  DATETIME,
    status       TEXT NOT NULL DEFAULT 'running'
);

CREATE TABLE IF NOT EXISTS sync_results (
    id          TEXT PRIMARY KEY,
    run_id      TEXT NOT NULL REFERENCES sync_runs(id) ON DELETE CASCADE,
    instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    config_type TEXT NOT NULL,
    status      TEXT NOT NULL,
    diff_json   TEXT,
    error_msg   TEXT,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS app_config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
