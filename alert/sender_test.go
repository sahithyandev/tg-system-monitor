package alert

import (
	"testing"
	"tg-system-monitor/db"
	"tg-system-monitor/detection"
	"time"
)

func TestSender(t *testing.T) {
	// Use in-memory database for testing
	database, err := db.Init(":memory:")
	if err != nil {
		t.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	// Create a mock telegram bot for testing
	mockBot := &MockTelegramBot{}

	// Create alert sender with short interval for testing
	sender := NewSender(database, mockBot, 100*time.Millisecond)

	// Test immediate alert sending
	alert := detection.Alert{
		Type:      detection.CPU,
		Severity:  detection.Warning,
		Value:     75.0,
		Threshold: 70.0,
		Message:   "CPU usage high",
	}

	err = sender.SendAlertImmediate(alert, "triggered")
	if err != nil {
		t.Errorf("Failed to send immediate alert: %v", err)
	}

	// Verify alert was sent
	if mockBot.sentAlerts != 1 {
		t.Errorf("Expected 1 alert to be sent, got %d", mockBot.sentAlerts)
	}
}

// MockTelegramBot implements the alert sending interface for testing
type MockTelegramBot struct {
	sentAlerts int
}

func (m *MockTelegramBot) SendAlert(alert detection.Alert, transition string) error {
	m.sentAlerts++
	return nil
}
