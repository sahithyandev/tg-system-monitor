package telegram

import (
	"fmt"
	"time"

	"tg-system-monitor/db"
	msg "tg-system-monitor/message"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// pingHandler handles /ping command
func pingHandler(database *db.DB, getLastMetricTime func() time.Time) func(b *gotgbot.Bot, ctx *ext.Context) error {
	return func(b *gotgbot.Bot, ctx *ext.Context) error {
		message := ctx.EffectiveMessage
		if message == nil {
			return nil
		}

		fmt.Println(msg.LogCompleted(msg.ComponentBot, fmt.Sprintf("/ping command from %s (@%s)", message.From.FirstName, message.From.Username)))

		// Check database connectivity
		dbStatus := "no"
		if err := database.Ping(); err != nil {
			fmt.Println(msg.LogFailed(msg.ComponentDatabase, "ping", err.Error()))
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
			fmt.Println(msg.LogFailed(msg.ComponentBot, "ping reply", err.Error()))
			return err
		}

		fmt.Println(msg.LogCompleted(msg.ComponentBot, fmt.Sprintf("ping reply sent to %s", message.From.FirstName)))
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

		fmt.Println(msg.LogCompleted(msg.ComponentBot, fmt.Sprintf("/whoami command from %s (@%s)", message.From.FirstName, message.From.Username)))

		// Get user from database
		user, err := database.GetUser(message.From.Id)
		if err != nil {
			fmt.Println(msg.LogFailed(msg.ComponentDatabase, "user lookup", err.Error()))
			_, err := message.Reply(b, msg.ErrorTemplate("User Retrieval Failed", "Unable to retrieve user information from database", "Please try again later"), nil)
			return err
		}

		// Determine user status
		var allowedStatus, alertsStatus string
		if user != nil {
			allowedStatus = "✅ Authenticated"
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
			fmt.Println(msg.LogFailed(msg.ComponentBot, "whoami reply", err.Error()))
			return err
		}

		fmt.Println(msg.LogCompleted(msg.ComponentBot, fmt.Sprintf("whoami reply sent to %s", message.From.FirstName)))
		return nil
	}
}

func helpHandler() func(b *gotgbot.Bot, ctx *ext.Context) error {
	return func(b *gotgbot.Bot, ctx *ext.Context) error {
		message := ctx.EffectiveMessage
		if message == nil {
			return nil
		}

		fmt.Println(msg.LogCompleted(msg.ComponentBot, fmt.Sprintf("/help command from %s (@%s)", message.From.FirstName, message.From.Username)))

		// Create help response
		response := `🤖 *Bot Commands Help*

📋 *Public Commands:*
/ping - Check bot health and database status
/whoami - Display your profile and authentication status
/help - Show this help message

🔐 *Authentication Commands:*
/join <password> - Authenticate with the bot using password
/leave - Remove yourself from the system

📊 *Authenticated Commands:*
/status - Show system metrics
/alerts <on|off> - Toggle alert notifications
/allow - Verify your authentication status
/disallow - Confirm authentication status

💡 *Usage Tips:*
• Use /join with the correct password to get access to restricted commands
• Type / in chat to see the command menu
• Use /whoami to check your current authentication status`

		_, err := message.Reply(b, response, &gotgbot.SendMessageOpts{
			ParseMode: "markdown",
		})
		if err != nil {
			fmt.Println(msg.LogFailed(msg.ComponentBot, "help reply", err.Error()))
			return err
		}

		fmt.Println(msg.LogCompleted(msg.ComponentBot, fmt.Sprintf("help reply sent to %s", message.From.FirstName)))
		return nil
	}
}
