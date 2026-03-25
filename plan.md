## Recommended shape

Each Linux machine runs its **own independent bot instance**.

Each instance:

* monitors only its own machine
* stores its own allowlist and settings in local SQLite
* sends alerts only for that machine
* responds to Telegram commands for that machine
* does not depend on any central coordinator

That matches your “one bot per instance” requirement exactly.

---

# 1. Functional scope

## Mandatory features

### Monitoring

* detect sustained high CPU usage
* detect sustained high memory usage
* detect low disk space
* optionally detect abnormal load average
* optionally detect swap pressure

### Telegram commands

* `/start` — intro and help
* `/join <password>` — join allowlist
* `/leave` — leave allowlist / disable alerts
* `/status` — current system resource summary
* `/top` — top CPU / memory consumers
* `/alerts on` — enable alert broadcasts
* `/alerts off` — disable alert broadcasts
* `/whoami` — show auth/subscription status
* `/ping` — health check

### Broadcast list

* maintain per-instance list of Telegram users allowed to receive alerts
* joining requires shared password
* only allowlisted users can request status/top data

---

# 2. Non-functional requirements

## Lightweight target

Aim for:

* single compiled binary
* RAM usage ideally under ~30–50 MB in normal idle state
* low CPU overhead from polling
* SQLite file only, no external DB
* no containers required
* no background web server unless needed for webhooks

## Deployment mode

Prefer:

* **Telegram long polling**
* `systemd` service
* local SQLite DB under `/var/lib/...` or `/opt/...`

Long polling is simpler and usually fine for one bot per server.


# 3. Suggested tech stack

## Libraries

* Telegram Bot API client for Go
* SQLite driver
* system metrics:

  * either standard Linux `/proc` parsing
  * or a lightweight library like `gopsutil`

### My recommendation

Use:

* **Go**
* **SQLite**
* **manual `/proc` reading for core metrics**
* minimal Telegram library

Reason:

* lowest dependency surface
* best control over resource usage
* avoids dragging in heavy abstraction layers

You only need:

* CPU
* memory
* load average
* disk usage
* uptime
* top processes

All of that is available from Linux procfs/sysfs plus `statfs`.

---

# 4. High-level architecture

## Internal modules

### 1) `main`

* load config
* initialize DB
* start bot loop
* start monitor loop
* handle graceful shutdown

### 2) `config`

Loads:

* Telegram bot token
* join password hash
* polling interval
* alert thresholds
* cooldowns
* paths
* admin IDs if any

### 3) `db`

SQLite access layer for:

* users
* subscriptions
* bot state
* alert cooldown state

### 4) `auth`

* verify `/join <password>`
* enforce allowlist on restricted commands

### 6) `detector`

Converts metric samples into alerts:

* sustained threshold detection
* cooldown suppression
* recovery detection

### 7) `telegram`

* poll updates
* route commands
* send direct replies
* broadcast alerts to all subscribed users

### 8) `formatter`

Build human-readable messages:

* status
* top processes
* alert text
* recovery text

---

# 5. SQLite schema

Use a very small schema.

## `users`

```sql
CREATE TABLE IF NOT EXISTS users (
    telegram_user_id INTEGER PRIMARY KEY,
    username TEXT,
    first_name TEXT,
    last_name TEXT,
    joined_at TEXT,
    is_allowed INTEGER NOT NULL DEFAULT 0,
    alerts_enabled INTEGER NOT NULL DEFAULT 1,
    last_seen_at TEXT
);
```

## `settings`

For instance-local persistent settings.

```sql
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

## `alert_state`

Tracks active alerts and cooldown timing.

```sql
CREATE TABLE IF NOT EXISTS alert_state (
    alert_key TEXT PRIMARY KEY,
    is_active INTEGER NOT NULL DEFAULT 0,
    active_since_unix INTEGER,
    last_triggered_unix INTEGER,
    last_recovered_unix INTEGER
);
```

## `metric_samples`

Optional. Keep only if you want short local history.

```sql
CREATE TABLE IF NOT EXISTS metric_samples (
    ts_unix INTEGER NOT NULL,
    cpu_percent REAL,
    mem_percent REAL,
    swap_percent REAL,
    disk_percent REAL,
    load1 REAL,
    load5 REAL,
    load15 REAL
);
```

If you want maximum lightness, you can skip `metric_samples` and keep recent samples in memory only.

## `failed_auth`

Optional for rate limiting `/join`.

```sql
CREATE TABLE IF NOT EXISTS failed_auth (
    telegram_user_id INTEGER PRIMARY KEY,
    fail_count INTEGER NOT NULL DEFAULT 0,
    last_fail_unix INTEGER
);
```

---

# 6. Monitoring model

## Polling interval

Recommended:

* every **15 seconds**

That is frequent enough for alerting but still light.

## Sampling windows

Do not alert on one bad sample.

Recommended defaults:

* CPU high: above 85% for 5 minutes
* memory high: above 90% for 3 minutes
* disk high: above 95% for 2 samples
* swap high: above 25% for 3 minutes
* load high: load1 greater than `num_cpu * 1.5` for 5 minutes

## Cooldowns

Per alert type:

* 30 minutes cooldown between repeated notifications for the same ongoing condition

## Recovery notifications

Send a recovery message when:

* alert was active
* metric goes back below a lower recovery threshold for enough samples

Example:

* CPU alert threshold: 85%
* CPU recovery threshold: 70%

This avoids flapping.

---

# 7. Metrics collection plan

## CPU usage

Read from `/proc/stat` and compute delta across samples.

Store:

* previous total CPU ticks
* previous idle ticks

Compute:

* `(delta_total - delta_idle) / delta_total * 100`

This is very lightweight and accurate.

## Memory usage

Use `/proc/meminfo`.

Prefer calculation:

* memory pressure based on `MemAvailable`

Formula:

* `used_percent = (MemTotal - MemAvailable) / MemTotal * 100`

This is better than naïve “used memory”.

## Swap

Also from `/proc/meminfo`:

* `SwapTotal`
* `SwapFree`

## Load average

Read from `/proc/loadavg`.

## Disk usage

Use `statfs` on configured mount points, at least `/`.

Optional later:

* support multiple mountpoints from config

## Uptime

Read from `/proc/uptime`.

## Top processes

For `/top` and for including context in alerts:

* scan `/proc/[pid]/stat` and `/proc/[pid]/status`
* compute process CPU usage from deltas, or simplify by showing memory-heavy processes + recent CPU snapshot
* keep this bounded and efficient

For first version, top processes can be:

* top by RSS memory
* plus top CPU consumers based on sampled deltas over last interval

---

# 8. Alert detection logic

Use an in-memory rolling window plus persisted alert state in SQLite.

## Example alert keys

* `cpu_high`
* `mem_high`
* `swap_high`
* `disk_root_high`
* `load_high`

## Per cycle

1. collect metrics
2. append to rolling window
3. evaluate thresholds
4. compare with `alert_state`
5. if trigger condition met and cooldown expired:

   * mark active
   * update timestamps
   * broadcast
6. if recovery condition met:

   * mark inactive
   * send recovery message

---

# 9. Telegram command plan

## Public commands

### `/start`

Returns:

* what this bot monitors
* how to join
* command list

### `/join <password>`

Behavior:

* only valid in private chat
* compare password hash with stored configured hash
* on success:

  * add/update user row
  * set `is_allowed=1`
  * set `alerts_enabled=1`

### `/leave`

* set `is_allowed=0`
* set `alerts_enabled=0`

## Restricted commands

Require `is_allowed=1`.

### `/status`

Return:

* hostname
* uptime
* CPU %
* memory %
* swap %
* disk usage for `/`
* load average
* alert states
* timestamp

### `/top`

Return:

* top N CPU processes
* top N memory processes

### `/alerts on`

Set `alerts_enabled=1`

### `/alerts off`

Set `alerts_enabled=0`

### `/whoami`

Return:

* allowed yes/no
* alerts enabled yes/no
* Telegram user ID
* username

### `/ping`

Return:

* bot alive
* last metric collection timestamp
* DB OK yes/no

---

# 10. Authentication design

## Password join flow

Store in config:

* **bcrypt hash** or **SHA-256 hash** of the join password

Do not store plain password in DB.

## Rules

* only allow `/join` in private chats
* rate limit failed join attempts
* optionally lock out for a few minutes after repeated failures

## Why this is enough

For a per-instance bot, this is a reasonable simple access control model.

---

# 11. Message design

## Status message example

```text
Host: web-01
Uptime: 4d 03h 15m

CPU: 22.4%
Memory: 61.8%
Swap: 0.0%
Disk /: 74.1%
Load avg: 0.84 0.92 0.88

Active alerts: none
Updated: 2026-03-25 14:10:03
```

## Alert message example

```text
ALERT: High CPU on web-01

CPU has remained above 85% for 5m
Current CPU: 93.2%
Load avg: 6.12 5.44 4.92
Top CPU:
1. java (pid 1234) 71.3%
2. nginx (pid 2441) 18.2%

Time: 2026-03-25 14:12:30
```

## Recovery message example

```text
RECOVERY: CPU normalized on web-01

CPU is back to 28.4%
Condition cleared after 17m
Time: 2026-03-25 14:29:12
```

---

# 12. Configuration plan

Use a small config file, environment variables, or both.

## Minimal config values

```toml
bot_token = "..."
join_password_hash = "..."
hostname_override = ""
poll_interval_seconds = 15

cpu_threshold_percent = 85
cpu_recovery_percent = 70
cpu_sustain_seconds = 300

mem_threshold_percent = 90
mem_recovery_percent = 80
mem_sustain_seconds = 180

swap_threshold_percent = 25
swap_recovery_percent = 10
swap_sustain_seconds = 180

disk_threshold_percent = 95
disk_recovery_percent = 90

alert_cooldown_seconds = 1800
top_process_count = 5
db_path = "/var/lib/telemon/bot.db"
```

Optional:

* list of monitored mountpoints
* admin Telegram IDs
* timezone override

---

# 13. Deployment plan

## Binary

Compile a single Linux binary:

* `telemon`

## Filesystem layout

Example:

```text
/opt/telemon/telemon
/etc/telemon/config.toml
/var/lib/telemon/bot.db
```

## systemd unit

```ini
[Unit]
Description=Telegram Linux Monitor Bot
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/opt/telemon/telemon --config /etc/telemon/config.toml
Restart=always
RestartSec=5
User=telemon
Group=telemon
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/telemon
CapabilityBoundingSet=
AmbientCapabilities=

[Install]
WantedBy=multi-user.target
```

## Service user

Create a dedicated user:

* `telemon`

Do not run as root unless absolutely necessary.

---

# 14. Resource minimization tactics

To stay lightweight:

## Avoid

* web dashboards
* Prometheus exporters
* large dependency trees
* container runtime overhead
* frequent DB writes for every tiny sample

## Prefer

* in-memory rolling windows
* DB writes only for:

  * user changes
  * alert state transitions
  * optional sparse history
* polling every 15s, not every 1s
* bounded process scans
* single-threaded design with small goroutines where needed

---

# 15. Logging plan

Use structured but simple logs.

Log:

* startup and config summary
* Telegram polling status
* command handling
* alert triggers
* alert recovery
* DB errors
* failed auth attempts

Do not log:

* raw join passwords
* secrets
* full bot token

Add log rotation via systemd journald or logrotate if file logging is used.

---

# 16. Failure handling

## Telegram API failure

* retry with backoff
* continue monitoring locally
* send pending future alerts when API returns, but do not try to replay huge history

## SQLite busy/locked

* set busy timeout
* keep transactions short
* use WAL mode if needed

## Process read errors

Some `/proc` entries will disappear during scan.

* ignore and continue

## Restart behavior

On restart:

* reload user allowlist from SQLite
* reload alert state
* begin fresh rolling windows
* do not immediately fire unless new sustain window is met

---

# 17. Security plan

## Secrets

* bot token in config file readable only by service user
* password stored as hash
* filesystem permissions locked down

## Bot exposure

* only private chats for auth and status
* optionally ignore group chats entirely

## Abuse prevention

* failed `/join` rate limiting
* optional admin-only commands if added later

---

# 18. Build phases

## Phase 1 — core bot skeleton

* config loader
* SQLite init
* Telegram polling
* `/start`, `/join`, `/leave`, `/whoami`

## Phase 2 — system metrics

* CPU, memory, disk, load, uptime
* `/status`

## Phase 3 — alert engine

* rolling windows
* thresholds
* cooldowns
* broadcasts
* recovery notifications

## Phase 4 — process insight

* `/top`
* include top offenders in alerts

## Phase 5 — hardening

* auth rate limit
* better logging
* WAL/busy timeout
* systemd hardening
* config validation

## Phase 6 — polish

* multiple mountpoints
* nicer formatting
* admin tools
* optional local history retention

---

# 19. Testing plan

## Unit tests

* config validation
* password verification
* threshold detection logic
* cooldown logic
* recovery logic
* formatting

## Integration tests

* SQLite schema init and migrations
* fake Telegram update routing
* monitor loop with simulated metrics

## Manual tests

* join with correct and incorrect password
* request `/status`
* induce CPU load and verify alert
* induce recovery and verify recovery message
* restart service and verify persistence

## My default recommendation if you want me to proceed without waiting

* Go
* SQLite
* long polling
* private-chat only
* `/`, CPU, memory, swap, load, uptime
* no local history table initially
* only these commands: `/start /join /leave /status /top /alerts /whoami /ping`
