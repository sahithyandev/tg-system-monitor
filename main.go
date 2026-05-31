package main

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tg-system-monitor/alert"
	"tg-system-monitor/api"
	"tg-system-monitor/auth"
	"tg-system-monitor/config"
	"tg-system-monitor/db"
	"tg-system-monitor/detection"
	"tg-system-monitor/message"
	"tg-system-monitor/metrics"
	"tg-system-monitor/telegram"
	"time"
)

// Version variables set by GoReleaser
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Check for subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "set-password":
			handleSetPassword()
			return
		case "version":
			handleVersion()
			return
		case "help", "-h", "--help":
			printHelp()
			return
		}
	}

	// Default behavior: run the monitor
	runMonitor()
}

func handleSetPassword() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter password for bot authentication: ")
	password, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("%s", message.LogFailed(message.ComponentMain, "password reading", err.Error()))
	}

	password = strings.TrimSpace(password)
	if password == "" {
		log.Fatal(message.PasswordEmpty)
	}

	fmt.Print("Confirm password: ")
	confirmPassword, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("%s", message.LogFailed(message.ComponentMain, "confirmation reading", err.Error()))
	}

	confirmPassword = strings.TrimSpace(confirmPassword)
	if password != confirmPassword {
		log.Fatal(message.PasswordMismatch)
	}

	// Hash the password
	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("%s", message.LogFailed(message.ComponentMain, "password hashing", err.Error()))
	}

	// Update only the password hash in config
	if err := config.UpdatePasswordHash(hash); err != nil {
		log.Fatalf("%s", message.LogFailed(message.ComponentMain, "password hash update", err.Error()))
	}

	fmt.Printf("%s\n", message.SuccessTemplate("Password Hash Updated", "Authentication password has been successfully configured"))
	fmt.Printf("Users can now authenticate using the password you set.\n")
	fmt.Printf("Hash: %s\n", hash)
}

func handleVersion() {
	if version != "dev" {
		fmt.Printf("tgsm version %s\n", version)
	} else {
		fmt.Printf("tgsm version %s (commit: %s)\n", version, commit)
	}
	fmt.Printf("Built: %s\n", date)
}

func printHelp() {
	fmt.Printf("Telegram System Monitor\n\n")
	fmt.Printf("Usage:\n")
	fmt.Printf("  %s [command]\n\n", os.Args[0])
	fmt.Printf("Commands:\n")
	fmt.Printf("  set-password    Set authentication password for restricted commands\n")
	fmt.Printf("  version         Show version information\n")
	fmt.Printf("  help, -h, --help  Show this help message\n\n")
	fmt.Printf("If no command is provided, the system monitor will start.\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  %s                # Start the monitor\n", os.Args[0])
	fmt.Printf("  %s set-password   # Set authentication password\n", os.Args[0])
	fmt.Printf("  %s version        # Show version information\n", os.Args[0])
}

func runMonitor() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("%s", message.LogFailed(message.ComponentConfig, "config loading", err.Error()))
	}

	// Display the finalized thresholds
	cfg.PrintThresholds()

	if cfg.BotToken == "" {
		log.Printf("%s", message.LogDisabled(message.ComponentBot, "bot_token not configured; running in collector-only mode"))
	}

	var volumePaths []string
	for _, v := range cfg.Monitors.Disk.Volumes {
		volumePaths = append(volumePaths, v.Path)
	}
	collector := metrics.NewCollector(volumePaths)

	fmt.Printf("%s\n", message.LogStarted(message.ComponentDatabase, cfg.DBPath))
	database, err := db.Init(cfg.DBPath)
	if err != nil {
		log.Fatalf("%s", message.LogFailed(message.ComponentDatabase, "initialization", err.Error()))
	}
	defer database.Close()

	// Initialize detection engine with config values and fallbacks
	m := cfg.Monitors
	detectionConfig := detection.DetectionConfig{
		CPU: detection.Threshold{
			Warning:  m.CPU.RecoveryPercent,  // Use recovery as warning threshold
			Critical: m.CPU.ThresholdPercent, // Use threshold as critical
		},
		Memory: detection.Threshold{
			Warning:  m.Memory.RecoveryPercent,
			Critical: m.Memory.ThresholdPercent,
		},
		Disk: detection.Threshold{
			Warning:  m.Disk.RecoveryPercent,
			Critical: m.Disk.ThresholdPercent,
		},
		Swap: detection.Threshold{
			Warning:  m.Swap.RecoveryPercent,
			Critical: m.Swap.ThresholdPercent,
		},
		Load1: detection.Threshold{
			Warning:  m.Load.Load1.Warning,
			Critical: m.Load.Load1.Critical,
		},
		Load5: detection.Threshold{
			Warning:  m.Load.Load5.Warning,
			Critical: m.Load.Load5.Critical,
		},
		Load15: detection.Threshold{
			Warning:  m.Load.Load15.Warning,
			Critical: m.Load.Load15.Critical,
		},
		WindowSecs:   m.CPU.SustainSeconds, // Use CPU sustain seconds as window
		CooldownSecs: cfg.AlertCooldownSeconds,
		Hysteresis:   cfg.Hysteresis,
	}
	for _, v := range m.Disk.Volumes {
		threshold := v.ThresholdPercent
		if threshold == 0 {
			threshold = m.Disk.ThresholdPercent
		}
		recovery := v.RecoveryPercent
		if recovery == 0 {
			recovery = m.Disk.RecoveryPercent
		}
		detectionConfig.Volumes = append(detectionConfig.Volumes, detection.VolumeThreshold{
			Path: v.Path,
			Threshold: detection.Threshold{
				Warning:  recovery,
				Critical: threshold,
			},
		})
	}
	detector := detection.NewDetectionEngine(database, detectionConfig)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize telegram bot (optional)
	var tgBot *telegram.Bot
	if cfg.BotToken != "" {
		fmt.Printf("%s\n", message.LogStarted(message.ComponentBot, "telegram bot"))
		tgBot, err = telegram.New(cfg, database)
		if err != nil {
			log.Fatalf("%s", message.LogFailed(message.ComponentBot, "initialization", err.Error()))
		}

		if err := tgBot.Start(ctx); err != nil {
			log.Fatalf("%s", message.LogFailed(message.ComponentBot, "startup", err.Error()))
		}

		// Initialize and start alert sender
		alertSender := alert.NewSender(database, tgBot, 30*time.Second)
		fmt.Printf("%s\n", message.LogStarted(message.ComponentAlert, "alert sender (30s interval)"))
		go alertSender.Start(ctx)
	}

	// Start optional metrics HTTP API
	if cfg.MetricsAPIAddr != "" {
		fmt.Printf("%s\n", message.LogStarted(message.ComponentAPI, "metrics HTTP API on "+cfg.MetricsAPIAddr))
		apiServer := api.NewServer(database, cfg.MetricsAPIAddr)
		go apiServer.Start(ctx)
	} else {
		log.Printf("%s", message.LogDisabled(message.ComponentAPI, "metrics_api_addr not configured; HTTP API disabled"))
	}

	fmt.Printf("%s\n", message.LogStarted(message.ComponentMetrics, fmt.Sprintf("metrics collection (interval: %ds)", cfg.PollInterval)))

	// Start data retention goroutine
	go func() {
		purge := func() {
			n, err := database.PurgeOldData(cfg.DataRetentionDays)
			if err != nil {
				log.Printf("%s", message.LogFailed(message.ComponentDatabase, "data retention purge", err.Error()))
				return
			}
			if n > 0 {
				log.Printf("%s", message.LogCompleted(message.ComponentDatabase, fmt.Sprintf("purged %d old rows (retention: %d days)", n, cfg.DataRetentionDays)))
			}
		}

		purge()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				purge()
			}
		}
	}()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start metrics collection in goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
		defer ticker.Stop()

		collect := func() {
			sample, err := collector.Collect()
			if err != nil {
				log.Printf("%s", message.LogFailed(message.ComponentMetrics, "collection", err.Error()))
				return
			}

			if err := database.SaveMetricSample(sample); err != nil {
				log.Printf("%s", message.LogFailed(message.ComponentDatabase, "metric save", err.Error()))
				return
			}

			// Run detection evaluation
			if err := detector.EvaluateDetections(); err != nil {
				log.Printf("%s", message.LogFailed(message.ComponentDetection, "evaluation", err.Error()))
			}

			// Update last metric time in bot
			if tgBot != nil {
				tgBot.UpdateLastMetricTime(sample.Timestamp)
			}

			msg := message.LogCompleted(message.ComponentMetrics, fmt.Sprintf("saved metric sample. CPU: %.2f%%, Mem: %.2f%%, Disk: %.2f%%",
				sample.CPUPercent, sample.MemPercent, sample.DiskPercent))
			fmt.Printf("%s\n", msg)
		}

		collect()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				collect()
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\nShutting down gracefully...")
	cancel()

	// Stop telegram bot
	if tgBot != nil {
		tgBot.Stop()
	}
	fmt.Println("Shutdown complete")
}
