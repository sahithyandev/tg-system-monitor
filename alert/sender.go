package alert

import (
	"context"
	"fmt"
	"time"

	"tg-system-monitor/db"
	"tg-system-monitor/detection"
	msg "tg-system-monitor/message"
)

type AlertSender interface {
	SendAlert(alert detection.Alert, transition string) error
}

type Sender struct {
	db          *db.DB
	alertSender AlertSender
	interval    time.Duration
	batchSize   int
	maxRetries  int
}

func NewSender(database *db.DB, alertSender AlertSender, interval time.Duration) *Sender {
	return &Sender{
		db:          database,
		alertSender: alertSender,
		interval:    interval,
		batchSize:   10, // Process up to 10 alerts per cycle
		maxRetries:  3,
	}
}

func (s *Sender) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	fmt.Println(msg.LogStarted(msg.ComponentAlert, fmt.Sprintf("alert sender (interval: %v)", s.interval)))

	for {
		select {
		case <-ctx.Done():
			fmt.Println(msg.LogCompleted(msg.ComponentAlert, "alert sender stopping"))
			return
		case <-ticker.C:
			s.processAlerts()
		}
	}
}

func (s *Sender) processAlerts() {
	// Get unsent alerts with retry logic
	var transitions []db.Alert
	var err error

	// Retry getting alerts with exponential backoff
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			delay := time.Duration(100*attempt) * time.Millisecond
			time.Sleep(delay)
		}

		transitions, err = s.db.GetUnsentAlerts(s.batchSize)
		if err == nil {
			break
		}

		// Check if it's a retryable error
		if isRetryableSQLError(err) {
			fmt.Println(msg.LogFailed(msg.ComponentAlert, fmt.Sprintf("getting unsent alerts (attempt %d/3)", attempt+1), err.Error()))
			continue
		}

		// Non-retryable error
		fmt.Println(msg.LogFailed(msg.ComponentAlert, "getting unsent alerts", err.Error()))
		return
	}

	if err != nil {
		fmt.Println(msg.LogFailed(msg.ComponentAlert, "getting unsent alerts after retries", err.Error()))
		return
	}

	if len(transitions) == 0 {
		return // No alerts to process
	}

	fmt.Println(msg.LogCompleted(msg.ComponentAlert, fmt.Sprintf("processing %d alert transitions", len(transitions))))

	for _, alert := range transitions {
		if err := s.sendAlert(alert); err != nil {
			fmt.Println(msg.LogFailed(msg.ComponentAlert, fmt.Sprintf("sending alert ID %d", alert.ID), err.Error()))
			continue
		}

		// Mark as sent with retry logic
		if err := s.markAlertSentWithRetry(alert.ID); err != nil {
			fmt.Println(msg.LogFailed(msg.ComponentAlert, fmt.Sprintf("marking alert ID %d as sent", alert.ID), err.Error()))
		}
	}
}

// markAlertSentWithRetry marks an alert as sent with retry logic
func (s *Sender) markAlertSentWithRetry(id int64) error {
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			delay := time.Duration(100*attempt) * time.Millisecond
			time.Sleep(delay)
		}

		if err := s.db.MarkAlertSent(id); err == nil {
			return nil
		} else if !isRetryableSQLError(err) {
			return err // Non-retryable error
		}
	}

	return fmt.Errorf("failed to mark alert %d as sent after 3 attempts", id)
}

// isRetryableSQLError checks if a database error is worth retrying
func isRetryableSQLError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// SQLite busy errors
	if contains(errStr, "database is locked") || contains(errStr, "SQLITE_BUSY") {
		return true
	}

	// Connection errors
	if contains(errStr, "connection") && (contains(errStr, "refused") || contains(errStr, "reset")) {
		return true
	}

	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (s *Sender) sendAlert(alert db.Alert) error {
	// Convert to detection.Alert for telegram bot
	detectionAlert := detection.Alert{
		Type:      detection.AlertType(alert.AlertType),
		Severity:  detection.Severity(alert.Severity),
		Value:     alert.Value,
		Threshold: alert.Threshold,
		Message:   alert.Message,
	}

	// Send via alert sender interface
	return s.alertSender.SendAlert(detectionAlert, alert.Transition)
}

// SendAlertImmediate sends an alert immediately without storing in database
// This can be used for critical system events that need immediate delivery
func (s *Sender) SendAlertImmediate(alert detection.Alert, transition string) error {
	fmt.Println(msg.LogStarted(msg.ComponentAlert, fmt.Sprintf("immediate alert: %s %s", alert.Severity, alert.Type)))
	return s.alertSender.SendAlert(alert, transition)
}
