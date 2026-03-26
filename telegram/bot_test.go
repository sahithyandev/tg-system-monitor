package telegram

import (
	"context"
	"testing"
	"time"

	"tg-system-monitor/config"
	"tg-system-monitor/db"
)

func TestBotCreation(t *testing.T) {
	cfg := &config.Config{
		BotToken: "invalid_token_for_testing",
	}

	// Create a temporary database for testing
	database, err := db.Init(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer database.Close()

	bot, err := New(cfg, database)
	if err != nil {
		// Expected to fail with invalid token, but should not panic
		t.Logf("Expected error with invalid token: %v", err)
		return
	}

	// If somehow created with invalid token, test cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := bot.Start(ctx); err != nil {
		t.Logf("Expected error when starting with invalid token: %v", err)
	}

	bot.Stop()
}

func TestBotStructure(t *testing.T) {
	cfg := &config.Config{
		BotToken: "test_token",
	}

	// Create a temporary database for testing
	database, err := db.Init(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer database.Close()

	bot, err := New(cfg, database)
	if err != nil {
		// Invalid token is expected
		t.Logf("Bot creation failed as expected: %v", err)
		return
	}

	if bot == nil {
		t.Error("Bot should not be nil even with invalid token")
	}

	if bot.GetBot() == nil {
		t.Error("GetBot() should not return nil")
	}
}
