package telegram

import (
	"fmt"
	"log"
	"time"

	"tg-system-monitor/db"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// echoHandler handles text messages by echoing them back
func echoHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	message := ctx.EffectiveMessage
	if message == nil {
		return nil
	}

	log.Printf("Received message from %s (@%s): %s",
		message.From.FirstName,
		message.From.Username,
		message.Text)

	// Echo the message back
	_, err := message.Reply(b, fmt.Sprintf("Echo: %s", message.Text), nil)
	if err != nil {
		log.Printf("Failed to send echo reply: %v", err)
		return err
	}

	log.Printf("Sent echo reply to %s", message.From.FirstName)
	return nil
}

// pingHandler handles /ping command
func pingHandler(database *db.DB, getLastMetricTime func() time.Time) func(b *gotgbot.Bot, ctx *ext.Context) error {
	return func(b *gotgbot.Bot, ctx *ext.Context) error {
		message := ctx.EffectiveMessage
		if message == nil {
			return nil
		}

		log.Printf("Received /ping command from %s (@%s)",
			message.From.FirstName,
			message.From.Username)

		// Check database connectivity
		dbStatus := "no"
		if err := database.Ping(); err != nil {
			log.Printf("Database ping failed: %v", err)
		} else {
			dbStatus = "yes"
		}

		// Get last metric collection timestamp
		lastMetricTime := getLastMetricTime()
		var lastMetricStr string
		if lastMetricTime.IsZero() {
			lastMetricStr = "never"
		} else {
			lastMetricStr = lastMetricTime.Format("2006-01-02 15:04:05 UTC")
		}

		// Create ping response
		response := fmt.Sprintf("Bot is running.\n\nLast Metric Collection: %s\nDatabase healthy?: %s",
			lastMetricStr, dbStatus)

		_, err := message.Reply(b, response, &gotgbot.SendMessageOpts{
			ParseMode: "markdown",
		})
		if err != nil {
			log.Printf("Failed to send ping reply: %v", err)
			return err
		}

		log.Printf("Sent ping reply to %s", message.From.FirstName)
		return nil
	}
}

// whoamiHandler handles /whoami command
func whoamiHandler(database *db.DB) func(b *gotgbot.Bot, ctx *ext.Context) error {
	return func(b *gotgbot.Bot, ctx *ext.Context) error {
		message := ctx.EffectiveMessage
		if message == nil {
			return nil
		}

		log.Printf("Received /whoami command from %s (@%s)",
			message.From.FirstName,
			message.From.Username)

		// Get user from database
		user, err := database.GetUser(message.From.Id)
		if err != nil {
			log.Printf("Failed to get user from database: %v", err)
			_, err := message.Reply(b, "Error retrieving user information.", nil)
			return err
		}

		// Determine user status
		var allowedStatus, alertsStatus string
		if user != nil {
			allowedStatus = "✅ Authenticated"
			if user.AuthCount > 0 {
				allowedStatus += fmt.Sprintf(" (%d authentications)", user.AuthCount)
			}
		} else {
			allowedStatus = "❌ Not authenticated"
		}

		if user != nil && user.AlertsEnabled {
			alertsStatus = "🔔 Enabled"
		} else {
			alertsStatus = "🔕 Disabled"
		}

		// Format username
		username := message.From.Username
		if username == "" {
			username = "none"
		}

		// Create whoami response
		response := fmt.Sprintf("👤 *Your Profile*\n\n🆔 **ID:** `%d`\n👋 **Username:** @%s\n🔐 **Access:** %s\n🔔 **Alerts:** %s",
			message.From.Id, username, allowedStatus, alertsStatus)

		_, err = message.Reply(b, response, &gotgbot.SendMessageOpts{
			ParseMode: "markdown",
		})
		if err != nil {
			log.Printf("Failed to send whoami reply: %v", err)
			return err
		}

		log.Printf("Sent whoami reply to %s", message.From.FirstName)
		return nil
	}
}
