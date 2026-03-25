package main

import (
	"fmt"
	"log"
	"os"
	"tg-system-monitor/metrics"
	"tg-system-monitor/formatter"
)

func main() {
	collector := metrics.NewCollector()
	hostname, _ := os.Hostname()

	fmt.Println("Collecting sample metrics...")
	sample, err := collector.Collect()
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		fmt.Println("\nFormatted Status Message:")
		fmt.Println("-------------------------")
		fmt.Println(formatter.FormatStatus(hostname, sample))
		fmt.Println("-------------------------")
	}

	fmt.Println("\nCollecting top processes...")
	procs, err := collector.GetTopProcesses(5)
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		fmt.Println("\nFormatted Top Processes:")
		fmt.Println("-------------------------")
		fmt.Println(formatter.FormatTop(procs))
		fmt.Println("-------------------------")
	}
}
