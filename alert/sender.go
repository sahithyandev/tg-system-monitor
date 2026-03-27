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
	// Get unsent alerts
	transitions, err := s.db.GetUnsentAlerts(s.batchSize)
	if err != nil {
		fmt.Println(msg.LogFailed(msg.ComponentAlert, "getting unsent alerts", err.Error()))
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

		// Mark as sent
		if err := s.db.MarkAlertSent(alert.ID); err != nil {
			fmt.Println(msg.LogFailed(msg.ComponentAlert, fmt.Sprintf("marking alert ID %d as sent", alert.ID), err.Error()))
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
	fmt.Println(msg.LogStarted(msg.ComponentAlert, fmt.Sprintf("immediate alert: %s %s", alert.Severity, alert.Type)))
	return s.alertSender.SendAlert(alert, transition)
}
