package alert

import (
	"context"
	"log"
	"time"

	"tg-system-monitor/db"
	"tg-system-monitor/detection"
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

	log.Printf("Alert sender started with interval: %v", s.interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Alert sender stopping...")
			return
		case <-ticker.C:
			s.processAlerts()
		}
	}
}

func (s *Sender) processAlerts() {
	// Get unsent alerts
	transitions, err := s.db.GetUnsentAlerts(s.batchSize)
	if err != nil {
		log.Printf("Error getting unsent alerts: %v", err)
		return
	}

	if len(transitions) == 0 {
		return // No alerts to process
	}

	log.Printf("Processing %d alert transitions", len(transitions))

	for _, alert := range transitions {
		if err := s.sendAlert(alert); err != nil {
			log.Printf("Failed to send alert ID %d: %v", alert.ID, err)
			continue
		}

		// Mark as sent
		if err := s.db.MarkAlertSent(alert.ID); err != nil {
			log.Printf("Failed to mark alert ID %d as sent: %v", alert.ID, err)
		}
	}
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
	log.Printf("Sending immediate alert: %s %s", alert.Severity, alert.Type)
	return s.alertSender.SendAlert(alert, transition)
}
