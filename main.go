package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tg-system-monitor/auth"
	"tg-system-monitor/config"
	"tg-system-monitor/db"
	"tg-system-monitor/metrics"
	"tg-system-monitor/telegram"
	"time"
)

func main() {
	// Check for subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "set-password":
			handleSetPassword()
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
		log.Fatalf("Error reading password: %v\n", err)
	}

	password = strings.TrimSpace(password)
	if password == "" {
		log.Fatal("Password cannot be empty")
	}

	fmt.Print("Confirm password: ")
	confirmPassword, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Error reading confirmation: %v\n", err)
	}

	confirmPassword = strings.TrimSpace(confirmPassword)
	if password != confirmPassword {
		log.Fatal("Passwords do not match")
	}

	// Hash the password
	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("Error hashing password: %v\n", err)
	}

	// Load existing config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v\n", err)
	}

	// Update password hash
	cfg.JoinPasswordHash = hash

	// Save config
	if err := cfg.Save(); err != nil {
		log.Fatalf("Error saving config: %v\n", err)
	}

	fmt.Printf("\n✅ Password hash updated successfully in config file!\n")
	fmt.Printf("Users can now authenticate using the password you set.\n")
	fmt.Printf("Hash: %s\n", hash)
}

func printHelp() {
	fmt.Printf("Telegram System Monitor\n\n")
	fmt.Printf("Usage:\n")
	fmt.Printf("  %s [command]\n\n", os.Args[0])
	fmt.Printf("Commands:\n")
	fmt.Printf("  set-password    Set authentication password for restricted commands\n")
	fmt.Printf("  help, -h, --help  Show this help message\n\n")
	fmt.Printf("If no command is provided, the system monitor will start.\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  %s                # Start the monitor\n", os.Args[0])
	fmt.Printf("  %s set-password   # Set authentication password\n", os.Args[0])
}

func runMonitor() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config Error: %v\n", err)
	}
	fmt.Printf("Config loaded: Poll Interval = %ds\n", cfg.PollInterval)

	// Validate bot token
	if cfg.BotToken == "" {
		log.Fatal("Bot token is required in config")
	}

	collector := metrics.NewCollector()

	fmt.Printf("Initializing Database at %s...\n", cfg.DBPath)
	database, err := db.Init(cfg.DBPath)
	if err != nil {
		log.Fatalf("DB Init Error: %v\n", err)
	}
	defer database.Close()

	// Initialize telegram bot
	fmt.Println("Initializing Telegram bot...")
	tgBot, err := telegram.New(cfg, database)
	if err != nil {
		log.Fatalf("Failed to initialize telegram bot: %v\n", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start telegram bot
	if err := tgBot.Start(ctx); err != nil {
		log.Fatalf("Failed to start telegram bot: %v\n", err)
	}

	fmt.Printf("Starting metrics collection loop (Interval: %ds)...\n", cfg.PollInterval)
	fmt.Println("Press Ctrl+C to stop")

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
				log.Printf("Collection Error: %v\n", err)
				return
			}

			if err := database.SaveMetricSample(sample); err != nil {
				log.Printf("DB Save Error: %v\n", err)
				return
			}

			// Update last metric time in bot
			tgBot.UpdateLastMetricTime(sample.Timestamp)

			fmt.Printf("[%s] Sample saved. CPU: %.2f%%, Mem: %.2f%%\n",
				sample.Timestamp.Format("15:04:05"), sample.CPUPercent, sample.MemPercent)
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
	tgBot.Stop()
	fmt.Println("Shutdown complete")
}
