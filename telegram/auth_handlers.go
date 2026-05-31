package telegram

import (
	"fmt"
	"log"
	"strings"
	"time"

	"tg-system-monitor/auth"
	"tg-system-monitor/config"
	"tg-system-monitor/db"
	"tg-system-monitor/formatter"
	msg "tg-system-monitor/message"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// authMiddleware creates a middleware that checks authentication before executing the handler
func authMiddleware(authManager *auth.AuthManager, handler func(b *gotgbot.Bot, ctx *ext.Context) error) func(b *gotgbot.Bot, ctx *ext.Context) error {
	return func(b *gotgbot.Bot, ctx *ext.Context) error {
		message := ctx.EffectiveMessage
		if message == nil {
			return nil
		}

		if message.Chat.Type != "private" {
			_, err := message.Reply(b, msg.AuthDeniedTemplate("The `/join` command can only be used in private chats"), &gotgbot.SendMessageOpts{
				ParseMode: "markdown",
			})
			if err != nil {
				fmt.Println(msg.LogFailed(msg.ComponentBot, "private chat denial message", err.Error()))
			}
			fmt.Println(msg.LogIgnored(msg.ComponentBot, "/join command", fmt.Sprintf("non-private chat (%s) by user %s (@%s)", message.Chat.Type, message.From.FirstName, message.From.Username)))
			return nil
		}

		// Extract command and password from message
		parts := strings.Fields(message.Text)
		if len(parts) < 1 {
			return nil
		}

		command := strings.TrimPrefix(parts[0], "/")
		password := ""
		if len(parts) > 1 {
			password = parts[1]
		}

		// Authenticate user
		result := authManager.AuthenticateUser(
			message.From.Id,
			message.From.Username,
			command,
			password,
		)

		if !result.Authorized {
			_, err := message.Reply(b, msg.AuthDeniedTemplate(result.Reason), &gotgbot.SendMessageOpts{
				ParseMode: "markdown",
			})
			if err != nil {
				fmt.Println(msg.LogFailed(msg.ComponentBot, "auth denial message", err.Error()))
			}
			return nil
		}

		// If this was a password authentication (user didn't exist before), create user record
		if result.Reason == "Authentication successful" && password != "" {
			if err := authManager.CreateAuthenticatedUser(
				message.From.Id,
				message.From.Username,
				message.From.FirstName,
				message.From.LastName,
			); err != nil {
				fmt.Println(msg.LogFailed(msg.ComponentAuth, "user record creation", err.Error()))
				_, err := message.Reply(b, msg.ErrorTemplate("Authentication Failed", "Unable to create user authentication record", "Please try again or contact administrator"), nil)
				return err
			}

			// Send authentication success message
			response := msg.AuthSuccessTemplate(message.From.FirstName, message.From.Username, message.From.Id)

			_, err := message.Reply(b, response, &gotgbot.SendMessageOpts{
				ParseMode: "markdown",
			})
			if err != nil {
				fmt.Println(msg.LogFailed(msg.ComponentBot, "auth success message", err.Error()))
			}
		}

		// Call the original handler
		return handler(b, ctx)
	}
}

// joinHandler handles /join command with authentication
func joinHandler(authManager *auth.AuthManager, database *db.DB) func(b *gotgbot.Bot, ctx *ext.Context) error {
	return authMiddleware(authManager, func(b *gotgbot.Bot, ctx *ext.Context) error {
		message := ctx.EffectiveMessage
		if message == nil {
			return nil
		}

		fmt.Println(msg.LogCompleted(msg.ComponentBot, fmt.Sprintf("/join command from %s (@%s) in private chat", message.From.FirstName, message.From.Username)))

		// Extract password from message to check if this is initial authentication
		parts := strings.Fields(message.Text)
		password := ""
		if len(parts) > 1 {
			password = parts[1]
		}

		// Only create/update user record if this is not the initial authentication
		// (authMiddleware already handled user creation for initial auth)
		if password == "" {
			// This is a re-authentication, update the user record
			user := &db.User{
				ID:            message.From.Id,
				Username:      message.From.Username,
				FirstName:     message.From.FirstName,
				LastName:      message.From.LastName,
				FirstAuthAt:   time.Now(),
				LastAuthAt:    time.Now(),
				AlertsEnabled: true,
				CreatedAt:     time.Now(),
			}

			if err := database.UpdateUser(user); err != nil {
				fmt.Println(msg.LogFailed(msg.ComponentDatabase, "user record update", err.Error()))
				_, err := message.Reply(b, msg.ErrorTemplate("User Registration Failed", "Unable to create or update user record", "Please try again later"), nil)
				return err
			}

			response := msg.AuthSuccessTemplate(message.From.FirstName, message.From.Username, message.From.Id)

			_, err := message.Reply(b, response, &gotgbot.SendMessageOpts{
				ParseMode: "markdown",
			})
			if err != nil {
				fmt.Println(msg.LogFailed(msg.ComponentBot, "join confirmation", err.Error()))
				return err
			}
		}

		fmt.Println(msg.LogCompleted(msg.ComponentAuth, fmt.Sprintf("user authenticated: %s (@%s)", message.From.FirstName, message.From.Username)))
		return nil
	})
}

// leaveHandler handles /leave command with authentication
func leaveHandler(authManager *auth.AuthManager, database *db.DB) func(b *gotgbot.Bot, ctx *ext.Context) error {
	return authMiddleware(authManager, func(b *gotgbot.Bot, ctx *ext.Context) error {
		message := ctx.EffectiveMessage
		if message == nil {
			return nil
		}

		fmt.Println(msg.LogCompleted(msg.ComponentBot, fmt.Sprintf("/leave command from %s (@%s)", message.From.FirstName, message.From.Username)))

		// Delete user from database
		if err := database.DeleteUser(message.From.Id); err != nil {
			fmt.Println(msg.LogFailed(msg.ComponentDatabase, "user deletion", err.Error()))
			_, replyErr := message.Reply(b, msg.ErrorTemplate("User Removal Failed", "Unable to delete user record from system", "Please try again later"), nil)
			return replyErr
		}

		response := msg.SuccessTemplate("User Removed", fmt.Sprintf("User %s (@%s) has been removed from the system", message.From.FirstName, message.From.Username))

		_, err := message.Reply(b, response, &gotgbot.SendMessageOpts{
			ParseMode: "markdown",
		})
		if err != nil {
			fmt.Println(msg.LogFailed(msg.ComponentBot, "leave confirmation", err.Error()))
			return err
		}

		fmt.Println(msg.LogCompleted(msg.ComponentDatabase, fmt.Sprintf("user deleted: %s (@%s)", message.From.FirstName, message.From.Username)))
		return nil
	})
}

// alertsHandler handles /alerts command with authentication
func alertsHandler(authManager *auth.AuthManager, database *db.DB) func(b *gotgbot.Bot, ctx *ext.Context) error {
	return authMiddleware(authManager, func(b *gotgbot.Bot, ctx *ext.Context) error {
		message := ctx.EffectiveMessage
		if message == nil {
			return nil
		}

		parts := strings.Fields(message.Text)
		if len(parts) < 2 {
			_, err := message.Reply(b, msg.ErrorTemplate("Invalid Arguments", "Usage: /alerts <on|off>", "Please specify 'on' or 'off'"), nil)
			return err
		}

		action := strings.ToLower(parts[1])
		if action != "on" && action != "off" {
			_, err := message.Reply(b, msg.ErrorTemplate("Invalid Arguments", "Usage: /alerts <on|off>", "Please specify 'on' or 'off'"), nil)
			return err
		}

		fmt.Println(msg.LogCompleted(msg.ComponentBot, fmt.Sprintf("/alerts command from %s (@%s): %s", message.From.FirstName, message.From.Username, action)))

		// Get or create user record
		user, err := database.GetUser(message.From.Id)
		if err != nil {
			fmt.Println(msg.LogFailed(msg.ComponentDatabase, "user lookup", err.Error()))
			_, err := message.Reply(b, msg.ErrorTemplate("User Retrieval Failed", "Unable to retrieve user information from database", "Please try again later"), nil)
			return err
		}

		// Create user record if it doesn't exist
		if user == nil {
			user = &db.User{
				ID:            message.From.Id,
				Username:      message.From.Username,
				FirstName:     message.From.FirstName,
				LastName:      message.From.LastName,
				FirstAuthAt:   time.Now(),
				LastAuthAt:    time.Now(),
				AlertsEnabled: (action == "on"),
				CreatedAt:     time.Now(),
			}
			if err := database.UpdateUser(user); err != nil {
				log.Printf("Failed to create user record: %v", err)
				_, err := message.Reply(b, "❌ Failed to create user record.", nil)
				return err
			}
		} else {
			// Update existing user's alert preference
			user.AlertsEnabled = (action == "on")
			user.LastAuthAt = time.Now()
			if err := database.UpdateUser(user); err != nil {
				log.Printf("Failed to update user record: %v", err)
				_, err := message.Reply(b, "❌ Failed to update user record.", nil)
				return err
			}
		}

		status := "🔕 Disabled"
		if user.AlertsEnabled {
			status = "🔔 Enabled"
		}

		response := fmt.Sprintf("🔔 *Alerts Setting*\n\n👤 **User:** %s (@%s)\n🔔 **Status:** %s",
			message.From.FirstName, message.From.Username, status)

		_, err = message.Reply(b, response, &gotgbot.SendMessageOpts{
			ParseMode: "markdown",
		})
		if err != nil {
			log.Printf("Failed to send alerts confirmation: %v", err)
			return err
		}

		log.Printf("Alerts %s for user %s (@%s)", action, message.From.FirstName, message.From.Username)
		return nil
	})
}

// statusHandler handles /status command with authentication
func statusHandler(database *db.DB, cfg *config.Config) func(b *gotgbot.Bot, ctx *ext.Context) error {
	return func(b *gotgbot.Bot, ctx *ext.Context) error {
		message := ctx.EffectiveMessage
		if message == nil {
			return nil
		}

		log.Printf("Received /status command from %s (@%s)",
			message.From.FirstName,
			message.From.Username)

		// Get latest metric from database
		sample, err := database.GetLatestMetric()
		if err != nil {
			log.Printf("Failed to get latest metric: %v", err)
			_, err := message.Reply(b, "Error retrieving system metrics.", nil)
			return err
		}

		if sample == nil {
			_, err := message.Reply(b, "No system metrics available yet.", nil)
			return err
		}

		// Format status message using the existing formatter
		response := formatter.FormatStatus(sample)

		_, err = message.Reply(b, response, nil)
		if err != nil {
			log.Printf("Failed to send status reply: %v", err)
			return err
		}

		log.Printf("Sent status reply to %s", message.From.FirstName)
		return nil
	}
}
