# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
# Build
go build -ldflags="-s -w" -o tgsm main.go

# Test (all packages)
go test ./...

# Test a single package
go test ./detection/...

# Run locally
./tgsm               # starts the monitor
./tgsm set-password  # set join password
./tgsm version
```

Config is read from `~/.config/tg-system-monitor/config.yml`. Copy `default-config.yml` there to start.

## Architecture

The app is a long-running Go service. `main.go:runMonitor()` wires all components together and drives them via goroutines and tickers.

**Data flow:**
1. `metrics.Collector` samples CPU/mem/swap/disk/load on a ticker (default 15s) — platform-specific implementations in `metrics/metrics_linux.go` and `metrics/metrics_darwin.go`.
2. Each sample is persisted to SQLite via `db.DB.SaveMetricSample`.
3. `detection.DetectionEngine.EvaluateDetections()` reads recent samples from the DB and writes `Alert` rows when thresholds are breached or recovered. Detection logic handles hysteresis, cooldowns, and sustained-high windows.
4. `alert.Sender` polls the DB every 30s for unsent alerts and dispatches them to authenticated Telegram users via `telegram.Bot.SendAlert`.

**Packages:**

| Package | Role |
|---------|------|
| `config` | Loads and validates `config.yml`; handles legacy flat keys via migration |
| `db` | SQLite wrapper (WAL mode). Tables: metric samples, users, alert states, alert queue |
| `metrics` | OS-level collection; `Collector` is platform-gated via build tags |
| `detection` | Threshold evaluation engine; alert state machine stored in DB |
| `alert` | Polls DB for unsent alerts; sends via the `AlertSender` interface |
| `telegram` | `gotgbot/v2` bot; registers handlers in `bot.go`, implements them in `handlers.go` and `auth_handlers.go` |
| `api` | Optional HTTP server exposing `GET /metrics` and `GET /health` |
| `auth` | bcrypt password hashing + `AuthManager` backed by DB |
| `formatter` | Formats metric values for Telegram messages |
| `message` | Shared log/status message templates |

**Optional modes:**
- `bot_token` omitted → collector-only mode (no Telegram, no alerts).
- `metrics_api_addr` set → HTTP API enabled alongside bot.

## Alerts pipeline detail

`detection.DetectionEngine` writes to the DB alert queue. `alert.Sender` reads that queue and calls `telegram.Bot.SendAlert`. The bot implements the `alert.AlertSender` interface. This decoupling means detection and delivery are independent — detection runs in the metrics goroutine; delivery runs on its own 30s ticker.

## Release

Releases are triggered by pushing a `v*` tag. GoReleaser (`.goreleaser.yml`) builds Linux/macOS amd64+arm64 binaries plus `.deb`/`.rpm` packages. Version variables (`version`, `commit`, `date`) are injected via `-ldflags` at build time by GoReleaser.

Versioning rules: MAJOR for breaking config/schema changes, MINOR for new user-visible features, PATCH for everything else.
