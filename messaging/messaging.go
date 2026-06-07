package messaging

import (
	"context"
	"fmt"
)

type Message struct {
	ChannelID string
	UserID    string
	Content   string
	ReplyTo   string
}

type MessageHandler func(msg Message) (string, error)

type Platform interface {
	Name() string
	Start(ctx context.Context, handler MessageHandler) error
	Stop() error
	SendMessage(channelID, content string) error
}

type Config struct {
	PlatformType    string
	BotToken        string
	AllowedUsers    []string
	AllowedChannels []string
	AllowedDir      string
	VoiceEnabled    bool
	VoiceDevice     string
}

func NewPlatform(cfg Config) (Platform, error) {
	switch cfg.PlatformType {
	case "discord":
		return NewDiscord(cfg)
	case "telegram":
		return NewTelegram(cfg)
	case "":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported messaging platform: %s", cfg.PlatformType)
	}
}
