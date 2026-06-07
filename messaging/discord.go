package messaging

import (
	"context"
	"fmt"
)

type DiscordBot struct {
	cfg Config
}

func NewDiscord(cfg Config) (*DiscordBot, error) {
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("discord bot token is required")
	}
	return &DiscordBot{cfg: cfg}, nil
}

func (d *DiscordBot) Name() string { return "discord" }

func (d *DiscordBot) Start(ctx context.Context, handler MessageHandler) error {
	fmt.Println("[Discord] Bot stub active — requires github.com/bwmarrin/discordgo to be vendored and imported")
	fmt.Println("[Discord] To enable: add discordgo to go.mod and implement the real client in this file")
	<-ctx.Done()
	return nil
}

func (d *DiscordBot) Stop() error { return nil }

func (d *DiscordBot) SendMessage(channelID, content string) error {
	fmt.Printf("[Discord] [%s] %s\n", channelID, content)
	return nil
}
