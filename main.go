package main

import (
	"fmt"
	"log"
	"time"
	"tg-system-monitor/metrics"
)

func main() {
	collector := metrics.NewCollector()

	fmt.Println("Collecting initial metrics (CPU will be 0 on first call)...")
	sample, err := collector.Collect()
	if err != nil {
		log.Printf("Error: %v (Note: this is expected on non-Linux systems)\n", err)
	} else {
		fmt.Printf("Initial Sample: %+v\n", sample)
	}

	time.Sleep(2 * time.Second)

	fmt.Println("Collecting second sample...")
	sample, err = collector.Collect()
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Second Sample: %+v\n", sample)
	}

	fmt.Println("\nTop 5 processes by memory:")
	procs, err := collector.GetTopProcesses(5)
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		for i, p := range procs {
			fmt.Printf("%d. %s (PID: %d, Memory: %d KB)\n", i+1, p.Name, p.PID, p.Memory)
		}
	}
}
