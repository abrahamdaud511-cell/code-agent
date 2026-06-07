package messaging

import (
	"context"
	"fmt"
)

type TelegramBot struct {
	cfg Config
}

func NewTelegram(cfg Config) (*TelegramBot, error) {
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}
	return &TelegramBot{cfg: cfg}, nil
}

func (t *TelegramBot) Name() string { return "telegram" }

func (t *TelegramBot) Start(ctx context.Context, handler MessageHandler) error {
	fmt.Println("[Telegram] Bot stub active — requires github.com/tucnak/telebot or similar to be vendored and imported")
	fmt.Println("[Telegram] To enable: add telebot to go.mod and implement the real client in this file")
	<-ctx.Done()
	return nil
}

func (t *TelegramBot) Stop() error { return nil }

func (t *TelegramBot) SendMessage(channelID, content string) error {
	fmt.Printf("[Telegram] [%s] %s\n", channelID, content)
	return nil
}
