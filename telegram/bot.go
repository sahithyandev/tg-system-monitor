package telegram

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"tg-system-monitor/auth"
	"tg-system-monitor/config"
	"tg-system-monitor/db"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
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
			log.Printf("Telegram dispatcher error: %v", err.Error())
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

	// Register ping command handler
	dispatcher.AddHandler(handlers.NewCommand("ping", pingHandler(database, b.GetLastMetricTime)))

	// Register whoami command handler
	dispatcher.AddHandler(handlers.NewCommand("whoami", whoamiHandler(database)))

	// Register status command handler
	dispatcher.AddHandler(handlers.NewCommand("status", statusHandler(database, cfg)))

	// Register restricted command handlers with authentication
	dispatcher.AddHandler(handlers.NewCommand("join", joinHandler(b.auth, database)))
	dispatcher.AddHandler(handlers.NewCommand("leave", leaveHandler(b.auth, database)))
	dispatcher.AddHandler(handlers.NewCommand("allow", allowHandler(b.auth, database)))
	dispatcher.AddHandler(handlers.NewCommand("disallow", disallowHandler(b.auth, database)))
	dispatcher.AddHandler(handlers.NewCommand("alerts", alertsHandler(b.auth, database)))

	// Register echo handler for all text messages
	dispatcher.AddHandler(handlers.NewMessage(message.Text, echoHandler))

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

	log.Printf("Telegram bot started successfully as @%s", b.bot.User.Username)

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
		log.Println("Telegram bot shutdown timeout")
	}
	log.Println("Telegram bot stopped")
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
