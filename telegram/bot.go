package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"tg-system-monitor/auth"
	"tg-system-monitor/config"
	"tg-system-monitor/db"
	"tg-system-monitor/detection"
	msg "tg-system-monitor/message"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
)

type Bot struct {
	bot            *gotgbot.Bot
	updater        *ext.Updater
	cancel         context.CancelFunc
	done           chan struct{}
	db             *db.DB
	auth           *auth.AuthManager
	lastMetricTime atomic.Value // stores time.Time
}

func New(cfg *config.Config, database *db.DB) (*Bot, error) {
	bot, err := gotgbot.NewBot(cfg.BotToken, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create updater and dispatcher
	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			fmt.Println(msg.LogFailed(msg.ComponentBot, "telegram dispatcher", err.Error()))
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
		Logger:      logger,
	})
	updater := ext.NewUpdater(dispatcher, &ext.UpdaterOpts{Logger: logger})

	// Create bot instance
	b := &Bot{
		bot:     bot,
		updater: updater,
		db:      database,
		auth:    auth.NewAuthManager(database, cfg.JoinPasswordHash),
		done:    make(chan struct{}),
	}
	b.lastMetricTime.Store(time.Time{}) // Initialize with zero time

	// Set bot commands menu
	if err := b.setBotCommands(); err != nil {
		fmt.Println(msg.LogFailed(msg.ComponentBot, "set commands", err.Error()))
	}

	// Register ping command handler
	dispatcher.AddHandler(handlers.NewCommand("ping", pingHandler(database, b.GetLastMetricTime)))

	// Register whoami command handler
	dispatcher.AddHandler(handlers.NewCommand("whoami", whoamiHandler(database)))

	// Register status command handler with authentication
	dispatcher.AddHandler(handlers.NewCommand("status", authMiddleware(b.auth, statusHandler(database, cfg))))

	// Register restricted command handlers with authentication
	dispatcher.AddHandler(handlers.NewCommand("join", joinHandler(b.auth, database)))
	dispatcher.AddHandler(handlers.NewCommand("leave", leaveHandler(b.auth, database)))
	dispatcher.AddHandler(handlers.NewCommand("alerts", alertsHandler(b.auth, database)))

	// Register help command handler
	dispatcher.AddHandler(handlers.NewCommand("help", helpHandler()))

	return b, nil
}

func (b *Bot) Start(ctx context.Context) error {
	ctx, b.cancel = context.WithCancel(ctx)

	// Start receiving updates
	err := b.updater.StartPolling(b.bot, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 9,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 10,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to start polling: %w", err)
	}

	fmt.Println(msg.LogCompleted(msg.ComponentBot, fmt.Sprintf("telegram bot started as @%s", b.bot.User.Username)))

	// Run updater in goroutine
	go func() {
		defer close(b.done)
		defer b.cancel()
		b.updater.Idle()
	}()

	return nil
}

func (b *Bot) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	// Stop the updater
	b.updater.Stop()
	// Wait for the bot goroutine to finish
	select {
	case <-b.done:
	case <-time.After(5 * time.Second):
		fmt.Println(msg.LogFailed(msg.ComponentBot, "shutdown timeout", "timeout reached"))
	}
	fmt.Println(msg.LogCompleted(msg.ComponentBot, "telegram bot stopped"))
}

func (b *Bot) GetBot() *gotgbot.Bot {
	return b.bot
}

func (b *Bot) UpdateLastMetricTime(t time.Time) {
	b.lastMetricTime.Store(t)
}

func (b *Bot) GetLastMetricTime() time.Time {
	return b.lastMetricTime.Load().(time.Time)
}

func (b *Bot) SendAlert(alert detection.Alert, transition string) error {
	// Get all users who have alerts enabled
	users, err := b.db.GetAllowedUsers()
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	// Format alert message
	alertMessage := msg.AlertTemplate(
		strings.ToUpper(string(alert.Type)),
		string(alert.Severity),
		strings.ToUpper(string(alert.Type)),
		fmt.Sprintf("%.2f/%.2f", alert.Value, alert.Threshold),
		transition,
	)

	// Send to all users with alerts enabled
	var sentCount int
	var errors []string

	for _, user := range users {
		if user.AlertsEnabled {
			_, err := b.bot.SendMessage(user.ID, alertMessage, &gotgbot.SendMessageOpts{
				ParseMode: "markdown",
			})
			if err != nil {
				errors = append(errors, fmt.Sprintf("user %d: %v", user.ID, err))
			} else {
				sentCount++
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s: %s", msg.LogFailed(msg.ComponentBot, "alert delivery", "multiple users failed"), strings.Join(errors, "; "))
	}
	if sentCount == 0 {
		return nil
	}

	fmt.Println(msg.LogCompleted(msg.ComponentBot, fmt.Sprintf("alert sent to %d users", sentCount)))
	return nil
}

// setBotCommands sets the bot's command menu using Telegram's SetMyCommands API
func (b *Bot) setBotCommands() error {
	commands := []gotgbot.BotCommand{
		{Command: "ping", Description: "Check bot health and database status"},
		{Command: "whoami", Description: "Display your profile and authentication status"},
		{Command: "help", Description: "Show all available commands"},
		{Command: "join", Description: "Authenticate with the bot using password"},
		{Command: "status", Description: "Show system metrics (requires authentication)"},
		{Command: "alerts", Description: "Toggle alert notifications (on/off)"},
		{Command: "leave", Description: "Remove yourself from the system"},
	}

	_, err := b.bot.SetMyCommands(commands, &gotgbot.SetMyCommandsOpts{
		Scope:        &gotgbot.BotCommandScopeDefault{},
		LanguageCode: "",
	})

	if err != nil {
		return fmt.Errorf("failed to set bot commands: %w", err)
	}

	fmt.Println(msg.LogCompleted(msg.ComponentBot, "bot commands menu set"))
	return nil
}
