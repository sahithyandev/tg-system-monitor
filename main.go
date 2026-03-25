package main

import (
	"fmt"
	"log"
	"tg-system-monitor/config"
	"tg-system-monitor/db"
	"tg-system-monitor/metrics"
	"time"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config Error: %v\n", err)
	}
	fmt.Printf("Config loaded: Poll Interval = %ds\n", cfg.PollInterval)

	collector := metrics.NewCollector()

	fmt.Printf("Initializing Database at %s...\n", cfg.DBPath)
	database, err := db.Init(cfg.DBPath)
	if err != nil {
		log.Fatalf("DB Init Error: %v\n", err)
	}
	defer database.Close()

	fmt.Printf("Starting metrics collection loop (Interval: %ds)...\n", cfg.PollInterval)
	fmt.Println("Press Ctrl+C to stop")

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

		fmt.Printf("[%s] Sample saved. CPU: %.2f%%, Mem: %.2f%%\n",
			sample.Timestamp.Format("15:04:05"), sample.CPUPercent, sample.MemPercent)
	}

	collect()

	for range ticker.C {
		collect()
	}
}
