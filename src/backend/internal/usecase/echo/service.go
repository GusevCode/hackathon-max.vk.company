package echo

import (
	"context"
	"fmt"
	"log/slog"
)

// Service contains the echo bot use case and has no dependency on MAX SDK types.
type Service struct {
	bot    BotGateway
	logger *slog.Logger
}

func NewService(bot BotGateway, logger *slog.Logger) *Service {
	return &Service{bot: bot, logger: logger}
}

func (s *Service) Run(ctx context.Context) error {
	info, err := s.bot.GetInfo(ctx)
	if err != nil {
		return fmt.Errorf("get MAX bot info: %w", err)
	}

	s.logger.Info("MAX bot connected", "name", info.Name, "username", info.Username, "user_id", info.ID)
	if info.Username != "" {
		s.logger.Info("MAX bot link", "url", "https://max.ru/"+info.Username)
	}

	for message := range s.bot.Messages(ctx) {
		if err := s.bot.SendMessage(ctx, message); err != nil {
			s.logger.Error("send echo", "error", err, "chat_id", message.ChatID)
		}
	}

	return ctx.Err()
}
