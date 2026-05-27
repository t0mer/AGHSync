# AGHSync

**AGHSync** keeps your [AdGuardHome](https://github.com/AdguardTeam/AdGuardHome) instances in sync automatically. Designate one instance as the **master** — AGHSync propagates its configuration to every **slave** instance on a schedule, on demand, via webhook, or whenever the AdGuardHome config file changes on disk.

---

## Screenshots

### Dashboard
![Dashboard](assets/screenshots/dashboard.png)

### Dark Mode
![Dark Mode](assets/screenshots/dark-mode.png)

### Instances
![Instances](assets/screenshots/instances.png)

### Sync Configuration (master)
![Sync Config](assets/screenshots/sync-config.png)

### Filesystem Watchdog
![Filesystem Watchdog](assets/screenshots/watchdog.png)

### Sync History
![History](assets/screenshots/history.png)

### Run Detail with Diff
![History Detail](assets/screenshots/history-detail.png)

### Notifications
![Notifications](assets/screenshots/notifications.png)

### Add Notification Channel
![Add Notification Channel](assets/screenshots/notifications-add.png)

### Settings
![Settings](assets/screenshots/settings.png)

---

## Features

### Instance Management
- Add, edit, and delete AdGuardHome instances
- Designate one instance as **Master** — promoting a slave automatically demotes the current master and transfers its sync configuration
- Connection test with live credential validation before saving
- TLS skip-verify option per instance for self-signed certificates
- **Duplicate prevention** — each address can only be added once; a clear error is shown if you try to add the same instance twice
- **Online/Offline status** — a colored dot per instance refreshes every 60 seconds
- **AGH version badge** — the running AdGuardHome version is shown per instance alongside the status dot
- **Per-instance last sync time** — the Instances table shows when each slave was last synced and whether it succeeded or failed, with a green/red indicator
- **Per-slave sync toggle** — each slave can be individually enabled or disabled; disabled slaves are skipped during sync

### Synchronisation
- **Granular sync config** — the master controls which AdGuardHome configuration types are pushed to slaves via per-type checkboxes:
  - `blocked_services`, `dhcp`, `dns`, `filtering`, `parental`, `rewrite`, `safebrowsing`, `safesearch`, `tls`
- **Scheduled sync** — user-configurable cron expression (e.g. `0 * * * *` for hourly)
- **Manual run** — trigger a sync instantly from the UI or via API
- **Sync on startup** — optional toggle to trigger a full sync automatically when AGHSync starts
- **Webhook trigger** — `POST /api/v1/webhook/sync` for external integrations (e.g. AdGuardHome post-update hooks)
- **Filesystem watchdog** — watches the AdGuardHome config file for changes and automatically triggers a sync when it is updated; supports Linux, Windows, and UNC paths; changes are debounced to handle atomic multi-step writes (works correctly inside Docker volumes)
- Sync runs concurrently across all slave instances

### Notifications
- **Notification channels** — receive a message after every sync run with a summary of what changed per instance
- **Three providers supported:**
  - **Shoutrrr** — Slack, Discord, Telegram, Gotify, SMTP, and [many more](https://containrrr.dev/shoutrrr/services/overview/)
  - **GreenAPI** — WhatsApp cloud messaging via [GreenAPI](https://green-api.com)
  - **WhatsApp Web** — self-hosted WhatsApp via [go-whatsapp-web-multidevice](https://github.com/aldinokemal/go-whatsapp-web-multidevice)
- Per-channel toggles for **notify on success** and **notify on failure / partial failure**
- Channels can be **enabled or disabled** without deleting them
- **Send Test** button fires a real message from the add/edit dialog before saving
- Channel credentials are stored **AES-256-GCM encrypted** in the database

### Dashboard
- Master instance summary, live sync status, and last run result
- Per-instance stats cards showing:
  - AdGuardHome version
  - Total DNS queries
  - Blocked by filters
  - Blocked malware / phishing
  - Average DNS processing time
- Stats refresh automatically every 60 seconds

### History & Diff
- Full sync run history with status (`Success`, `Partial Failure`, `Error`), trigger source, start time, and duration
- Per-run detail view with a result row per config type and per slave
- **Change indicator** — an amber icon flags config types where a change was actually applied
- **LCS-based diff viewer** — click any row to expand a green/red unified diff showing exactly what changed, with `+N / -N` summary badge

### Settings
- **UI Authentication** — enable/disable Basic Auth for the web interface; username and password set via the UI
- **API Token** — generate a secure token for protecting the REST API; shown once and never stored in plaintext
- **Backup & Restore** — export all settings to a JSON file and import to fully restore a previous state. The backup includes:
  - All instances and their credentials (encrypted)
  - Per-instance sync configuration
  - Watchdog settings
  - Scheduler configuration
  - Notification channels (encrypted)
  - UI auth and API token

### API
- Every UI action has an equivalent REST endpoint
- Swagger / OpenAPI documentation served at `/api/docs`
- Token-authenticated (`X-API-Token` header) or Basic Auth
- See `/api/docs` for the full endpoint reference

### Dark Mode
- System-preference-aware dark/light toggle in the navbar

---

## Quick Start

### Docker (recommended)

```bash
docker run -d \
  --name aghsync \
  -p 8080:8080 \
  -v aghsync-data:/app/data \
  -e AGHSYNC_DATA=/app/data \
  techblog/aghsync:latest
```

Open [http://localhost:8080](http://localhost:8080)

### Docker Compose

```yaml
services:
  aghsync:
    image: techblog/aghsync:latest
    ports:
      - "8080:8080"
    volumes:
      - aghsync-data:/app/data
    environment:
      AGHSYNC_DATA: /app/data
      LOG_LEVEL: info

volumes:
  aghsync-data:
```

```bash
docker compose up -d
```

### Binary

Download the binary for your platform from the [Releases](../../releases) page.

```bash
# Linux / macOS
chmod +x aghsync-linux-amd64
./aghsync-linux-amd64

# Windows
aghsync-windows-amd64.exe
```

---

## Configuration

### CLI Flags

| Flag | Description |
|---|---|
| `--port <n>` | Listening port. Default: `8080`. Overridden by `AGHSYNC_PORT`. |
| `--log-level <level>` | `debug` / `info` / `warning` / `error`. Default: `warning`. |
| `--reset-password` | Interactively reset the UI login password. |
| `--service <action>` | Manage the OS service: `install` / `uninstall` / `start` / `stop` / `restart`. |

### Environment Variables

| Variable | Description |
|---|---|
| `AGHSYNC_PORT` | Server port — takes precedence over `--port`. |
| `LOG_LEVEL` | Log level — takes precedence over `--log-level`. |
| `AGHSYNC_DATA` | Directory for `aghsync.db`. Default: current working directory. |

Port resolution order (highest → lowest): `AGHSYNC_PORT` → `--port` → built-in default (`8080`).

---

## Filesystem Watchdog

The watchdog monitors the AdGuardHome configuration file for changes and triggers an immediate sync whenever it is updated. It is designed to work correctly with AdGuardHome's atomic-write pattern (write to a temp file → rename), which is the default on all platforms and inside Docker containers.

**Setup:** go to **Settings → Watchdog**, enable it, and enter the full path to the AdGuardHome `AdGuardHome.yaml` file.

**Docker example** — if AdGuardHome writes its config to `/opt/adguardhome/conf/AdGuardHome.yaml` inside its container, mount that same path into AGHSync:

```yaml
services:
  aghsync:
    image: techblog/aghsync:latest
    volumes:
      - aghsync-data:/app/data
      - /opt/adguardhome/conf:/opt/adguardhome/conf:ro
    environment:
      AGHSYNC_DATA: /app/data
```

Then set the watchdog path to `/opt/adguardhome/conf/AdGuardHome.yaml`.

**Supported path formats:**

| Format | Example |
|---|---|
| Linux absolute | `/etc/adguardhome/AdGuardHome.yaml` |
| Windows absolute | `C:\AdGuardHome\AdGuardHome.yaml` |
| UNC (network share) | `\\server\share\AdGuardHome.yaml` |

---

## Notification Channels

After every sync run AGHSync can send a message to one or more notification channels. The message includes the run result and a per-instance summary of what changed.

### Shoutrrr

[Shoutrrr](https://containrrr.dev/shoutrrr/services/overview/) supports Slack, Discord, Telegram, Gotify, SMTP e-mail, ntfy, and many more via a single URL scheme.

```
slack://token@channel
discord://token@id
telegram://token@telegram?chats=@channel
gotify://hostname/token
smtp://user:password@host:port/?from=from@example.com&to=to@example.com
```

### GreenAPI (WhatsApp)

Send WhatsApp messages via [GreenAPI](https://green-api.com). Requires a GreenAPI account and an active instance.

| Field | Description |
|---|---|
| Instance ID | Found in the GreenAPI console |
| Token | API token for the instance |
| Recipient Phone | International format, no `+` or spaces (e.g. `14085551234`) |
| API URL *(optional)* | Cluster-specific URL from the GreenAPI console; leave blank to use the default |

### WhatsApp Web (self-hosted)

Send WhatsApp messages via a self-hosted [go-whatsapp-web-multidevice](https://github.com/aldinokemal/go-whatsapp-web-multidevice) instance.

| Field | Description |
|---|---|
| Base URL | URL of your go-whatsapp-web-multidevice instance (e.g. `http://localhost:3000`) |
| Recipient Phone | International format, no `+` or spaces |
| Username *(optional)* | Basic Auth username if your instance is protected |
| Password *(optional)* | Basic Auth password |

---

## Supported Platforms

| OS | Architecture |
|---|---|
| Linux | amd64, arm64, armv7, armv6, 386 |
| macOS | amd64 (Intel), arm64 (Apple Silicon) |
| Windows | amd64, arm64 |

Docker images: `linux/amd64`, `linux/arm64`, `linux/arm/v7`

---

## API

The full REST API is documented at **`/api/docs`** (Swagger UI).

Key endpoints:

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/instances` | List all instances |
| `POST` | `/api/v1/instances` | Add an instance |
| `PUT` | `/api/v1/instances/{id}` | Update an instance |
| `DELETE` | `/api/v1/instances/{id}` | Remove an instance |
| `PUT` | `/api/v1/instances/{id}/promote` | Promote slave to master |
| `GET` | `/api/v1/instances/{id}/sync-config` | Get master sync config |
| `PUT` | `/api/v1/instances/{id}/sync-config` | Update master sync config |
| `GET` | `/api/v1/instances/statuses` | Online/offline status and AGH version for all instances |
| `GET` | `/api/v1/instances/last-sync` | Last sync time and status per instance |
| `PUT` | `/api/v1/instances/{id}/sync-enabled` | Enable or disable sync for a slave |
| `GET` | `/api/v1/instances/{id}/stats` | DNS stats for one instance |
| `POST` | `/api/v1/sync/run` | Trigger a manual sync |
| `GET` | `/api/v1/sync/status` | Current and last run status |
| `PUT` | `/api/v1/sync/schedule` | Update the cron schedule |
| `POST` | `/api/v1/webhook/sync` | Webhook trigger |
| `GET` | `/api/v1/history` | List sync runs |
| `GET` | `/api/v1/history/{runId}` | Run detail with per-config diffs |
| `GET` | `/api/v1/notifications` | List notification channels |
| `POST` | `/api/v1/notifications` | Add a notification channel |
| `PUT` | `/api/v1/notifications/{id}` | Update a notification channel |
| `DELETE` | `/api/v1/notifications/{id}` | Remove a notification channel |
| `POST` | `/api/v1/notifications/test` | Send a test message |
| `GET` | `/api/v1/settings` | Get application settings |
| `PUT` | `/api/v1/settings/ui-auth` | Enable/disable UI auth |
| `POST` | `/api/v1/settings/api-token` | Generate API token |
| `DELETE` | `/api/v1/settings/api-token` | Remove API token |
| `PUT` | `/api/v1/settings/watchdog` | Configure filesystem watchdog |
| `PUT` | `/api/v1/settings/sync-on-startup` | Enable/disable sync on startup |
| `GET` | `/api/v1/backup/export` | Download settings backup |
| `POST` | `/api/v1/backup/restore` | Restore from backup |

**Authentication:**
- No token configured → all `/api/v1` requests pass through (bootstrap mode)
- Token configured → requests must include `X-API-Token: <token>` **or** valid Basic Auth credentials (when UI auth is enabled)

---

## Development

```bash
# Prerequisites: Go 1.22+, Node 20+

# Start backend + frontend with hot reload
./scripts/dev.sh

# Run tests
go test ./...

# Build all release targets → dist/
./scripts/build.sh

# Build Docker image
docker build -t aghsync .
```

---

## License

[MIT](LICENSE)
