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

## Author

Sahithyan K. (https://sahithyan.dev)