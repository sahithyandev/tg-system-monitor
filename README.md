# tg-system-monitor

A simple and lightweight system monitor that interfaces through Telegram. Written in Go.

## Installation

### Quick Install (Linux — recommended)

```sh
curl -fsSL https://raw.githubusercontent.com/sahithyandev/tg-system-monitor/main/scripts/install.sh | sudo bash
```

Detects your architecture and package manager, installs the appropriate `.deb` or `.rpm`
package, creates the `tgsm` system user, and enables the service on boot.

### Debian / Ubuntu

Download the `.deb` from the [releases page](https://github.com/sahithyandev/tg-system-monitor/releases) and install it:

```sh
sudo apt install ./tgsm_*_linux_amd64.deb     # x86-64
sudo apt install ./tgsm_*_linux_arm64.deb     # ARM64
```

### RHEL / Fedora / CentOS

Download the `.rpm` from the [releases page](https://github.com/sahithyandev/tg-system-monitor/releases):

```sh
sudo dnf install ./tgsm_*_linux_amd64.rpm     # x86-64
sudo dnf install ./tgsm_*_linux_arm64.rpm     # ARM64
```

### Manual (raw binary)

Download the binary for your platform from the [releases page](https://github.com/sahithyandev/tg-system-monitor/releases),
place it at `/usr/bin/tgsm`, and follow the post-install steps below manually
(create `tgsm` user, data dir, systemd unit).

### Post-install setup

After installation via any method:

1. **Edit the config**:
   ```
   /var/lib/tgsm/.config/tg-system-monitor/config.yml
   ```
   - To enable the Telegram bot, set `bot_token` to the token you got from [@BotFather](https://t.me/botfather). If omitted, the monitor runs in collector-only mode — metrics are still collected and stored, but no Telegram bot or alerts are active.

2. **Set the join password** (the password users type to authenticate with the bot — only needed if using the Telegram bot):
   ```sh
   sudo -u tgsm HOME=/var/lib/tgsm tgsm set-password
   ```

3. **Start the service**:
   ```sh
   sudo systemctl start tgsm
   sudo journalctl -u tgsm -f   # tail the logs
   ```

A Telegram user can then join with `/join <password>` in a private chat with the bot.

---

## Development

### Build

```sh
go build -ldflags="-s -w" -o tgsm main.go
```

### Test

```sh
go test ./...
```

## How the Bot Works

The system monitor operates as a continuously running service that collects system metrics, evaluates them against configurable thresholds, and provides real-time monitoring through Telegram.

### Architecture

1. **Metrics Collection**: The bot collects system metrics at regular intervals (default: 15 seconds)
2. **Detection Engine**: Evaluates metrics against warning and critical thresholds with sophisticated logic
3. **Alert System**: Sends notifications to authenticated users when thresholds are breached
4. **Telegram Interface**: Provides commands for monitoring and management

### Metrics Sampled

The bot continuously monitors the following system metrics:

- **CPU Usage**: Current CPU utilization percentage
- **Memory Usage**: RAM utilization percentage  
- **Swap Usage**: Swap space utilization percentage
- **Disk Usage**: Disk space utilization percentage
- **System Load**: 1-minute, 5-minute, and 15-minute load averages
- **System Uptime**: Current system uptime in seconds

### Detection Logic

The detection engine uses a multi-layered approach for intelligent alerting:

#### Threshold-Based Alerts
- **Warning Level**: Triggers when metrics exceed warning thresholds
- **Critical Level**: Triggers when metrics exceed critical thresholds
- **Recovery Detection**: Alerts when metrics fall below recovery thresholds with hysteresis

#### Advanced Detection Features
- **Sustained High Usage**: Detects when metrics remain high for extended periods (configurable windows)
- **Load Spike Detection**: Identifies sudden increases in system load
- **Resource Pressure**: Alerts when multiple metrics are simultaneously elevated
- **Hysteresis**: Prevents alert flapping by requiring metrics to drop below recovery thresholds
- **Cooldown Periods**: Suppresses duplicate alerts for configurable time periods

#### Default Thresholds
- **CPU**: Warning 70%, Critical 85% (sustained for 300 seconds)
- **Memory**: Warning 80%, Critical 90% (sustained for 180 seconds)  
- **Disk**: Warning 90%, Critical 95%
- **Swap**: Warning 10%, Critical 25% (sustained for 180 seconds)
- **Load1**: Warning 2.0, Critical 4.0
- **Load5**: Warning 1.5, Critical 3.0
- **Load15**: Warning 1.0, Critical 2.0

## Telegram Commands

### Public Commands
- `/ping` - Check bot status and last metric collection time
- `/whoami` - Display your user profile and authentication status
- `/help` - Show all available commands

### Authenticated Commands
- `/join <password>` - Authenticate with the bot (only works in private chats)
- `/status` - View current system metrics and status
- `/alerts <on|off>` - Enable or disable alert notifications
- `/leave` - Remove your account from the system

### Usage Examples
```
/ping                    # Check if bot is running
/help                    # Show all available commands
/join secretpassword     # Authenticate with password
/status                  # View current system metrics
/alerts on              # Enable alert notifications
/alerts off             # Disable alert notifications
/whoami                 # Check your user status
/leave                  # Remove your account
```

## Configuration

The bot uses a YAML configuration file located at `~/.config/tg-system-monitor/config.yml`. Copy `default-config.yml` as a starting point. Key settings:

| Key                      | Default                               | Description                                                                                       |
| ------------------------ | ------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `bot_token`              | `""`                                  | Telegram bot token from [@BotFather](https://t.me/botfather). Omit to run in collector-only mode. |
| `poll_interval_seconds`  | `15`                                  | How often metrics are sampled (seconds).                                                          |
| `alert_cooldown_seconds` | `1800`                                | Minimum time between repeated alerts for the same metric.                                         |
| `top_process_count`      | `5`                                   | Number of top processes shown in status output.                                                   |
| `db_path`                | `~/.config/tg-system-monitor/tgsm.db` | SQLite database path.                                                                             |
| `data_retention_days`    | `30`                                  | How long metric samples are kept.                                                                 |
| `hysteresis`             | `5.0`                                 | Recovery buffer (%). A metric must drop this far below `recovery_percent` before an alert clears. |
| `hostname_override`      | `""`                                  | Override the hostname shown in messages.                                                          |
| `metrics_api_addr`       | `""`                                  | Address to expose the HTTP metrics API (e.g. `127.0.0.1:9090`). Empty = disabled.                 |

### Monitor thresholds

Each metric is configured under `monitors.<metric>`:

```yaml
sustain_seconds: 300          # detection window: threshold must be exceeded for this long before alerting

monitors:
  cpu:
    threshold_percent: 85.0   # critical alert when CPU exceeds this
    recovery_percent: 70.0    # alert clears below (recovery_percent - hysteresis)

  memory:
    threshold_percent: 90.0
    recovery_percent: 80.0

  swap:
    threshold_percent: 25.0
    recovery_percent: 10.0

  disk:
    threshold_percent: 95.0
    recovery_percent: 90.0
    volumes:                  # optional extra mount points to monitor
      - path: /data
        threshold_percent: 95.0
        recovery_percent: 90.0

  load:
    load1:
      warning: 2.0
      critical: 4.0
    load5:
      warning: 1.5
      critical: 3.0
    load15:
      warning: 1.0
      critical: 2.0
```

## Metrics HTTP API

`tg-system-monitor` can expose the latest collected metrics over HTTP so other services
can query them programmatically. The feature is **disabled by default**; enable it by
setting `metrics_api_addr` in your config:

```yaml
# Bind to localhost only (recommended for host-local consumers):
metrics_api_addr: "127.0.0.1:9090"

# Or expose on all interfaces (use firewall rules to restrict access):
# metrics_api_addr: "0.0.0.0:9090"
```

### Endpoints

| Method | Path               | Description                            |
| ------ | ------------------ | ------------------------------------- |
| `GET`  | `/metrics`         | Latest metric sample as JSON          |
| `GET`  | `/metrics/history` | Downsampled metric history for charts |
| `GET`  | `/health`          | Database liveness check               |

### `GET /metrics` — response shape

```json
{
  "timestamp": 1748691600,
  "uptime_seconds": 86400.0,
  "cpu_percent": 12.5,
  "mem_percent": 54.3,
  "swap_percent": 0.0,
  "disk_percent": 61.2,
  "load1": 0.42,
  "load5": 0.38,
  "load15": 0.31,
  "volumes": [
    { "path": "/data", "percent": 73.1 }
  ]
}
```

Returns `503` with `{"error": "..."}` if no sample has been collected yet (within the
first poll interval after startup).

### `GET /metrics/history` — time-series for charts

Use this to render graphs of resource usage over a time window — an hour, a day, or up
to your full retention period. Instead of returning every raw sample (which at the 15s
poll rate is ~5,760 points per day), the endpoint **downsamples**: it splits the range
into fixed time buckets and returns one point per bucket with the `avg` and `max` of
each metric in that bucket. `avg` gives a clean trend line; `max` preserves short spikes
that averaging would otherwise hide (e.g. a brief disk-full or CPU pin).

**Query parameters:**

| Param    | Default        | Description                                                            |
| -------- | -------------- | -------------------------------------------------------------------- |
| `from`   | now − 1 hour   | Start of the window, Unix seconds.                                    |
| `to`     | now            | End of the window, Unix seconds.                                      |
| `bucket` | auto           | Seconds per bucket. Omit to let the server pick (~2,000 points for the range, never finer than the poll interval). |

The response is rejected with `400` if the requested range and bucket would produce
more than 5,000 points — widen `bucket` (or narrow the range) and retry. History only
goes back as far as `data_retention_days`; older samples are purged.

```json
{
  "from": 1748605200,
  "to": 1748691600,
  "bucket_seconds": 60,
  "points": [
    {
      "timestamp": 1748605200,
      "cpu_percent":  { "avg": 12.5, "max": 41.0 },
      "mem_percent":  { "avg": 54.3, "max": 55.1 },
      "swap_percent": { "avg": 0.0,  "max": 0.0  },
      "disk_percent": { "avg": 61.2, "max": 61.2 },
      "load1":  { "avg": 0.42, "max": 1.10 },
      "load5":  { "avg": 0.38, "max": 0.71 },
      "load15": { "avg": 0.31, "max": 0.44 },
      "volumes": [
        { "path": "/data", "percent": { "avg": 73.1, "max": 73.4 }, "total_bytes": 512000000000 }
      ]
    }
  ]
}
```

Points are ordered oldest first. Buckets with no samples are omitted (gaps in the data
show as gaps in the array). `timestamp` is the start of the bucket.

### Examples

```sh
curl -s http://127.0.0.1:9090/metrics | jq
curl -s http://127.0.0.1:9090/health

# History — last 24 hours, server-picked resolution:
curl -s "http://127.0.0.1:9090/metrics/history?from=$(date -d '24 hours ago' +%s)" | jq

# History — last 30 days at 1-hour resolution:
curl -s "http://127.0.0.1:9090/metrics/history?from=$(date -d '30 days ago' +%s)&bucket=3600" | jq
```

## Release Process

This project uses automated releases through GitHub Actions and GoReleaser.

### Automated Releases

Releases are automatically triggered when:
- A new tag is pushed matching the pattern `v*` (e.g., `v1.0.0`, `v1.2.3`)
- A manual workflow dispatch is triggered with a specified tag

### Creating a Release

1. Tag-based Release (Recommended)
   ```sh
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. Manual Release
   - Go to the Actions tab in GitHub
   - Select the "Release" workflow
   - Click "Run workflow"
   - Optionally specify a tag name

### What Gets Released

The automated release process:
- Runs all tests to ensure code quality
- Builds cross-platform binaries using GoReleaser (Linux amd64/arm64, macOS amd64/arm64)
- Produces `.deb` and `.rpm` packages for Linux
- Generates a GitHub release with all assets and a `checksums.txt`

### Versioning

This project follows [Semantic Versioning](https://semver.org/) (`MAJOR.MINOR.PATCH`) after v2.0.0.

Here are the rules followed:
- **Does this break anything for existing users?**   
   If yes, bump **MAJOR** and reset MINOR and PATCH to 0.  
   Examples: Renamed or removed config keys, changed database schema requiring migration, dropped distro/arch support.
- **Does this add anything a user can see or use?**   
   If yes, bump **MINOR** and reset PATCH to 0.   
   Examples: New Telegram command, new optional config field, new metric being monitored.
- **Otherwise**   
   Bump **PATCH**.   
   Examples: Bug fix, internal refactor, CI/workflow change, documentation update.

### GoReleaser Configuration

The project uses GoReleaser for consistent cross-platform builds. The configuration is defined in `.goreleaser.yml`.

## Author

Sahithyan K. (https://sahithyan.dev)