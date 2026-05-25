# AGHSync

Keep your AdGuardHome instances in sync automatically.

AGHSync designates one AdGuardHome instance as **master** and propagates its configuration to any number of **child** instances on a schedule or on demand.

## Quick start

```bash
docker run -p 8080:8080 -v aghsync-data:/app/data t0mer/aghsync
```

Open http://localhost:8080

## CLI flags

| Flag | Description |
|---|---|
| `--port <n>` | Listening port. Default: `8080`. Overridden by `AGHSYNC_PORT`. |
| `--log-level <level>` | `debug` / `info` / `warning` / `error`. Default: `warning`. |
| `--reset-password` | Reset the UI login password. |
| `--service <action>` | Manage the OS service: `install` / `uninstall` / `start` / `stop` / `restart`. |

## Environment variables

| Variable | Description |
|---|---|
| `AGHSYNC_PORT` | Server port — takes precedence over `--port`. |
| `LOG_LEVEL` | Log level — takes precedence over `--log-level`. |
| `AGHSYNC_DATA` | Directory for `aghsync.db`. Default: current directory. |

## Development

```bash
# Install hot-reload tool
go install github.com/air-verse/air@latest

# Start backend + frontend with hot reload
./scripts/dev.sh
```

## Building

```bash
# Cross-compile all targets → dist/
./scripts/build.sh

# Docker
docker build -t aghsync .
```
