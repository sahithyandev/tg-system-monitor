# tg-system-monitor

A simple and lightweight system monitor that interfaces through Telegram. Written in Go.

## Commands

### Build

```sh
go build -ldflags="-s -w" -o tgsm main.go
```

### Test

```sh
go test ./...
```

The above command will run tests all packages.

## Setup 

### Database

The bot uses SQLite for data storage, which is automatically configured on first run. The database is stored at `~/.config/tg-system-monitor/tgsm.db` by default. Recent improvements include enhanced concurrent access handling and automatic retry logic for database operations, ensuring reliable performance under high load.

### Telegram
 
1. Create a Telegram bot using @BotFather.
2. Copy the bot token (looks like `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`)
3. Replace `YOUR_BOT_TOKEN_HERE` with your actual bot token in the config file (located at `~/.config/tg-system-monitor/config.yml`).
4. Set the password for the bot using `set-password` command.

After that, the bot will connect and run on the bot.

A user can join the bot by using the `/join` command with the password.

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

### Authenticated Commands
- `/join <password>` - Authenticate with the bot (only works in private chats)
- `/status` - View current system metrics and status
- `/alerts <on|off>` - Enable or disable alert notifications
- `/leave` - Remove your account from the system
- `/allow` - Verify your authentication status
- `/disallow` - Verify your authentication status

### Usage Examples
```
/ping                    # Check if bot is running
/join secretpassword     # Authenticate with password
/status                  # View current system metrics
/alerts on              # Enable alert notifications
/alerts off             # Disable alert notifications
/whoami                 # Check your user status
/leave                  # Remove your account
```

## Configuration

The bot uses a YAML configuration file located at `~/.config/tg-system-monitor/config.yml`. Key settings include:

- `poll_interval_seconds`: Metrics collection interval
- `cpu_threshold_percent`: CPU critical threshold
- `mem_threshold_percent`: Memory critical threshold  
- `disk_threshold_percent`: Disk critical threshold
- `alert_cooldown_seconds`: Time between duplicate alerts
- `hysteresis`: Recovery buffer percentage

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
- Builds cross-platform binaries using GoReleaser
- Creates Docker images and pushes to GitHub Container Registry (ghcr.io)
- Generates a GitHub release with assets
- Creates checksums for all binaries

### GoReleaser Configuration

The project uses GoReleaser for consistent cross-platform builds. The configuration is defined in `.goreleaser.yml`.

## Author

Sahithyan K. (https://sahithyan.dev)