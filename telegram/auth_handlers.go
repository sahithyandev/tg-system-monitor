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

		// Update user's last seen time
		user, err := authManager.IsUserAllowed(message.From.Id)
		if err == nil && user {
			// Update last seen time in database
			existingUser, err := authManager.GetDatabase().GetUser(message.From.Id)
			if err == nil && existingUser != nil {
				existingUser.LastSeenAt = time.Now()
				authManager.GetDatabase().UpdateUser(existingUser)
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

		log.Printf("Received /join command from %s (@%s)",
			message.From.FirstName, message.From.Username)

		// Add user to allowlist
		err := authManager.AddUserToAllowlist(
			message.From.Id,
			message.From.Username,
			message.From.FirstName,
			message.From.LastName,
		)
		if err != nil {
			log.Printf("Failed to add user to allowlist: %v", err)
			_, err := message.Reply(b, "❌ Failed to add you to the allowlist.", nil)
			return err
		}

		response := fmt.Sprintf("✅ *Successfully joined*\n\n👤 **User:** %s (@%s)\n🆔 **ID:** `%d`\n🔐 **Status:** Authorized",
			message.From.FirstName, message.From.Username, message.From.Id)

		_, err = message.Reply(b, response, &gotgbot.SendMessageOpts{
			ParseMode: "markdown",
		})
		if err != nil {
			log.Printf("Failed to send join confirmation: %v", err)
			return err
		}

		log.Printf("User %s (@%s) joined successfully", message.From.FirstName, message.From.Username)
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

		// Remove user from allowlist
		err := authManager.RemoveUserFromAllowlist(message.From.Id)
		if err != nil {
			log.Printf("Failed to remove user from allowlist: %v", err)
			_, err := message.Reply(b, "❌ Failed to remove you from the allowlist.", nil)
			return err
		}

		response := fmt.Sprintf("👋 *Successfully left*\n\n👤 **User:** %s (@%s)\n🆔 **ID:** `%d`\n🔐 **Status:** No longer authorized",
			message.From.FirstName, message.From.Username, message.From.Id)

		_, err = message.Reply(b, response, &gotgbot.SendMessageOpts{
			ParseMode: "markdown",
		})
		if err != nil {
			log.Printf("Failed to send leave confirmation: %v", err)
			return err
		}

		log.Printf("User %s (@%s) left successfully", message.From.FirstName, message.From.Username)
		return nil
	})
}

// allowHandler handles /allow command with authentication (admin only)
func allowHandler(authManager *auth.AuthManager, database *db.DB) func(b *gotgbot.Bot, ctx *ext.Context) error {
	return authMiddleware(authManager, func(b *gotgbot.Bot, ctx *ext.Context) error {
		message := ctx.EffectiveMessage
		if message == nil {
			return nil
		}

		parts := strings.Fields(message.Text)
		if len(parts) < 2 {
			_, err := message.Reply(b, "Usage: /allow <user_id> [username]", nil)
			return err
		}

		// Parse user ID
		var userID int64
		_, err := fmt.Sscanf(parts[1], "%d", &userID)
		if err != nil {
			_, err := message.Reply(b, "❌ Invalid user ID format", nil)
			return err
		}

		username := ""
		if len(parts) > 2 {
			username = strings.TrimPrefix(parts[2], "@")
		}

		log.Printf("Received /allow command for user %d from %s (@%s)",
			userID, message.From.FirstName, message.From.Username)

		// Add user to allowlist
		err = authManager.AddUserToAllowlist(userID, username, "", "")
		if err != nil {
			log.Printf("Failed to add user %d to allowlist: %v", userID, err)
			_, err := message.Reply(b, "❌ Failed to add user to allowlist.", nil)
			return err
		}

		response := fmt.Sprintf("✅ *User Allowed*\n\n🆔 **User ID:** `%d`\n👤 **Username:** @%s\n🔐 **Status:** Authorized",
			userID, username)

		_, err = message.Reply(b, response, &gotgbot.SendMessageOpts{
			ParseMode: "markdown",
		})
		if err != nil {
			log.Printf("Failed to send allow confirmation: %v", err)
			return err
		}

		log.Printf("User %d (@%s) was allowed by %s (@%s)",
			userID, username, message.From.FirstName, message.From.Username)
		return nil
	})
}

// disallowHandler handles /disallow command with authentication (admin only)
func disallowHandler(authManager *auth.AuthManager, database *db.DB) func(b *gotgbot.Bot, ctx *ext.Context) error {
	return authMiddleware(authManager, func(b *gotgbot.Bot, ctx *ext.Context) error {
		message := ctx.EffectiveMessage
		if message == nil {
			return nil
		}

		parts := strings.Fields(message.Text)
		if len(parts) < 2 {
			_, err := message.Reply(b, "Usage: /disallow <user_id>", nil)
			return err
		}

		// Parse user ID
		var userID int64
		_, err := fmt.Sscanf(parts[1], "%d", &userID)
		if err != nil {
			_, err := message.Reply(b, "❌ Invalid user ID format", nil)
			return err
		}

		log.Printf("Received /disallow command for user %d from %s (@%s)",
			userID, message.From.FirstName, message.From.Username)

		// Remove user from allowlist
		err = authManager.RemoveUserFromAllowlist(userID)
		if err != nil {
			log.Printf("Failed to remove user %d from allowlist: %v", userID, err)
			_, err := message.Reply(b, "❌ Failed to remove user from allowlist.", nil)
			return err
		}

		response := fmt.Sprintf("🚫 *User Disallowed*\n\n🆔 **User ID:** `%d`\n🔐 **Status:** No longer authorized",
			userID)

		_, err = message.Reply(b, response, &gotgbot.SendMessageOpts{
			ParseMode: "markdown",
		})
		if err != nil {
			log.Printf("Failed to send disallow confirmation: %v", err)
			return err
		}

		log.Printf("User %d was disallowed by %s (@%s)",
			userID, message.From.FirstName, message.From.Username)
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

		// Get user from database
		user, err := database.GetUser(message.From.Id)
		if err != nil {
			log.Printf("Failed to get user from database: %v", err)
			_, err := message.Reply(b, "❌ Error retrieving user information.", nil)
			return err
		}

		if user == nil {
			_, err := message.Reply(b, "❌ User not found in database.", nil)
			return err
		}

		// Update alerts setting
		user.AlertsEnabled = (action == "on")
		user.LastSeenAt = time.Now()
		err = database.UpdateUser(user)
		if err != nil {
			log.Printf("Failed to update user alerts setting: %v", err)
			_, err := message.Reply(b, "❌ Failed to update alerts setting.", nil)
			return err
		}

		status := "🔕 Disabled"
		if user.AlertsEnabled {
			status = "🔔 Enabled"
		}

		response := fmt.Sprintf("🔔 *Alerts Updated*\n\n👤 **User:** %s (@%s)\n🔔 **Status:** %s",
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
