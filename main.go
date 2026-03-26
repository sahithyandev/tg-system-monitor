package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"tg-system-monitor/config"
	"tg-system-monitor/db"
	"tg-system-monitor/metrics"
	"tg-system-monitor/telegram"
	"time"
)

func main() {
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
