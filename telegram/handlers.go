package telegram

import (
	"fmt"
	"log"

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
