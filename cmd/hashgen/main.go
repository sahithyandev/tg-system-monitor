package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"tg-system-monitor/auth"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter password to hash: ")
	password, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		os.Exit(1)
	}

	password = strings.TrimSpace(password)
	if password == "" {
		fmt.Println("Password cannot be empty")
		os.Exit(1)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		fmt.Printf("Error hashing password: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nGenerated Argon2 hash:\n%s\n\n", hash)
	fmt.Printf("Add this to your config.yml as:\n")
	fmt.Printf("join_password_hash: %s\n", hash)
}
