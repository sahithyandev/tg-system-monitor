package main

import (
	"fmt"
	"log"
	"os"
	"time"
	"tg-system-monitor/metrics"
	"tg-system-monitor/formatter"
	"tg-system-monitor/db"
)

func main() {
	// 1. Metrics and Formatter test
	collector := metrics.NewCollector()
	hostname, _ := os.Hostname()

	fmt.Println("Collecting sample metrics...")
	sample, err := collector.Collect()
	if err != nil {
		log.Printf("Metrics Error: %v\n", err)
	} else {
		fmt.Println("\nFormatted Status Message:")
		fmt.Println("-------------------------")
		fmt.Println(formatter.FormatStatus(hostname, sample))
		fmt.Println("-------------------------")
	}

	// 2. Database test
	fmt.Println("\nTesting Database...")
	dbPath := "test.db"
	defer os.Remove(dbPath)

	database, err := db.Init(dbPath)
	if err != nil {
		log.Fatalf("DB Init Error: %v\n", err)
	}
	defer database.Close()

	testUser := &db.User{
		ID:            12345,
		Username:      "testuser",
		FirstName:     "Test",
		JoinedAt:      time.Now(),
		IsAllowed:     true,
		AlertsEnabled: true,
		LastSeenAt:    time.Now(),
	}

	if err := database.UpdateUser(testUser); err != nil {
		log.Printf("UpdateUser Error: %v\n", err)
	}

	retrievedUser, err := database.GetUser(12345)
	if err != nil {
		log.Printf("GetUser Error: %v\n", err)
	} else if retrievedUser != nil {
		fmt.Printf("Retrieved User: %+v\n", retrievedUser)
	}

	if err := database.SetSetting("test_key", "test_value"); err != nil {
		log.Printf("SetSetting Error: %v\n", err)
	}

	val, err := database.GetSetting("test_key")
	if err != nil {
		log.Printf("GetSetting Error: %v\n", err)
	} else {
		fmt.Printf("Retrieved Setting: %s\n", val)
	}
}
