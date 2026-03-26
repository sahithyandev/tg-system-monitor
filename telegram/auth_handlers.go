package telegram

import (
	"fmt"
	"log"
	"strings"
	"time"

	"tg-system-monitor/auth"
	"tg-system-monitor/db"

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
			_, err := message.Reply(b, fmt.Sprintf("🚫 *Access Denied*\n\n%s", result.Reason), &gotgbot.SendMessageOpts{
				ParseMode: "markdown",
			})
			if err != nil {
				log.Printf("Failed to send auth denial message: %v", err)
			}
			return nil
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

		log.Printf("Received /join command from %s (@%s)",
			message.From.FirstName, message.From.Username)

		// Create or update user record in database
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
			log.Printf("Failed to create/update user record: %v", err)
			_, err := message.Reply(b, "❌ Failed to create user record.", nil)
			return err
		}

		response := fmt.Sprintf("✅ *Authentication Successful*\n\n👤 **User:** %s (@%s)\n🆔 **ID:** `%d`\n🔐 **Status:** Authorized to use restricted commands",
			message.From.FirstName, message.From.Username, message.From.Id)

		_, err := message.Reply(b, response, &gotgbot.SendMessageOpts{
			ParseMode: "markdown",
		})
		if err != nil {
			log.Printf("Failed to send join confirmation: %v", err)
			return err
		}

		log.Printf("User %s (@%s) authenticated successfully (record created/updated)", message.From.FirstName, message.From.Username)
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

		log.Printf("Received /leave command from %s (@%s)",
			message.From.FirstName, message.From.Username)

		// Update user's last authentication time
		user, err := database.GetUser(message.From.Id)
		if err != nil {
			log.Printf("Failed to get user info: %v", err)
			_, err := message.Reply(b, "❌ Error retrieving user information.", nil)
			return err
		}

		if user != nil {
			user.LastAuthAt = time.Now()
			if err := database.UpdateUser(user); err != nil {
				log.Printf("Failed to update user auth record: %v", err)
				_, err := message.Reply(b, "❌ Failed to update authentication record.", nil)
				return err
			}
		}

		response := fmt.Sprintf("👋 *Session Ended*\n\n👤 **User:** %s (@%s)\n🆔 **ID:** `%d`\n🔐 **Status:** Authentication session completed",
			message.From.FirstName, message.From.Username, message.From.Id)

		_, err = message.Reply(b, response, &gotgbot.SendMessageOpts{
			ParseMode: "markdown",
		})
		if err != nil {
			log.Printf("Failed to send leave confirmation: %v", err)
			return err
		}

		log.Printf("User %s (@%s) session ended", message.From.FirstName, message.From.Username)
		return nil
	})
}

// allowHandler handles /allow command with authentication (now just confirms authentication)
func allowHandler(authManager *auth.AuthManager, database *db.DB) func(b *gotgbot.Bot, ctx *ext.Context) error {
	return authMiddleware(authManager, func(b *gotgbot.Bot, ctx *ext.Context) error {
		message := ctx.EffectiveMessage
		if message == nil {
			return nil
		}

		log.Printf("Received /allow command from %s (@%s)",
			message.From.FirstName, message.From.Username)

		// Get user info for response
		user, err := database.GetUser(message.From.Id)
		if err != nil {
			log.Printf("Failed to get user info: %v", err)
			_, err := message.Reply(b, "❌ Error retrieving user information.", nil)
			return err
		}

		response := fmt.Sprintf("✅ *Authentication Verified*\n\n👤 **User:** %s (@%s)\n🔐 **Status:** Authenticated for restricted commands\n\nℹ️ *Note:* Allowlist system has been removed. Any user with the correct password can access restricted commands.",
			message.From.FirstName, message.From.Username)

		_, err = message.Reply(b, response, &gotgbot.SendMessageOpts{
			ParseMode: "markdown",
		})
		if err != nil {
			log.Printf("Failed to send allow confirmation: %v", err)
			return err
		}

		log.Printf("User %s (@%s) verified authentication", message.From.FirstName, message.From.Username)
		return nil
	})
}

// disallowHandler handles /disallow command with authentication (now just confirms authentication)
func disallowHandler(authManager *auth.AuthManager, database *db.DB) func(b *gotgbot.Bot, ctx *ext.Context) error {
	return authMiddleware(authManager, func(b *gotgbot.Bot, ctx *ext.Context) error {
		message := ctx.EffectiveMessage
		if message == nil {
			return nil
		}

		log.Printf("Received /disallow command from %s (@%s)",
			message.From.FirstName, message.From.Username)

		// Get user info for response
		user, err := database.GetUser(message.From.Id)
		if err != nil {
			log.Printf("Failed to get user info: %v", err)
			_, err := message.Reply(b, "❌ Error retrieving user information.", nil)
			return err
		}

		response := fmt.Sprintf("🔒 *Authentication Confirmed*\n\n👤 **User:** %s (@%s)\n🔐 **Status:** Authenticated for restricted commands\n\nℹ️ *Note:* Allowlist system has been removed. Any user with the correct password can access restricted commands.",
			message.From.FirstName, message.From.Username)

		_, err = message.Reply(b, response, &gotgbot.SendMessageOpts{
			ParseMode: "markdown",
		})
		if err != nil {
			log.Printf("Failed to send disallow confirmation: %v", err)
			return err
		}

		log.Printf("User %s (@%s) confirmed authentication", message.From.FirstName, message.From.Username)
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
			_, err := message.Reply(b, "Usage: /alerts <on|off>", nil)
			return err
		}

		action := strings.ToLower(parts[1])
		if action != "on" && action != "off" {
			_, err := message.Reply(b, "❌ Invalid action. Use 'on' or 'off'", nil)
			return err
		}

		log.Printf("Received /alerts command from %s (@%s): %s",
			message.From.FirstName, message.From.Username, action)

		// Get or create user record
		user, err := database.GetUser(message.From.Id)
		if err != nil {
			log.Printf("Failed to get user info: %v", err)
			_, err := message.Reply(b, "❌ Error retrieving user information.", nil)
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
