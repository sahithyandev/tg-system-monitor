package main

import (
	"fmt"
	"log"
	"tg-system-monitor/db"
	"tg-system-monitor/metrics"
	"time"
)

func main() {
	collector := metrics.NewCollector()

	fmt.Println("Initializing Database...")
	dbPath := "telemon.db"

	database, err := db.Init(dbPath)
	if err != nil {
		log.Fatalf("DB Init Error: %v\n", err)
	}
	defer database.Close()

	fmt.Println("Starting metrics collection loop (Interval: 15s)...")
	fmt.Println("Press Ctrl+C to stop")

	ticker := time.NewTicker(15 * time.Second)
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

		fmt.Printf("[%s] Sample saved. CPU: %.1f%%, Mem: %.1f%%\n",
			sample.Timestamp.Format("15:04:05"), sample.CPUPercent, sample.MemPercent)
	}

	// Collect first sample immediately
	collect()

	for range ticker.C {
		collect()
	}
}
