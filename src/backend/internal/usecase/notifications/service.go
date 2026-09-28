package notifications

import (
	"context"
	"log/slog"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type Service struct {
	subscriber Subscriber
	sender     Sender
	logger     *slog.Logger
}

func NewService(subscriber Subscriber, sender Sender, logger *slog.Logger) *Service {
	return &Service{subscriber: subscriber, sender: sender, logger: logger}
}

// Run consumes broker events and translates them into MAX direct messages.
// This remains in the same binary as the control module, but has an explicit
// module boundary and can later be extracted into a separate worker.
func (s *Service) Run(ctx context.Context) error {
	events, err := s.subscriber.Subscribe(ctx)
	if err != nil {
		return err
	}
	for notification := range events {
		if notification.RecipientUserID == 0 {
			s.logger.Warn("skip notification without recipient", "notification_id", notification.ID)
			continue
		}
		message := domain.OutgoingMessage{
			UserID: notification.RecipientUserID,
			Text:   notification.Text,
			Photos: notification.Photos,
		}
		_, err := s.sender.Send(ctx, message)
		if err != nil {
			s.logger.Warn("send notification", "error", err, "notification_id", notification.ID, "kind", notification.Kind, "user_id", notification.RecipientUserID)
			if len(message.Photos) > 0 {
				message.Photos = nil
				if _, fallbackErr := s.sender.Send(ctx, message); fallbackErr != nil {
					s.logger.Warn("send notification fallback", "error", fallbackErr, "notification_id", notification.ID, "user_id", notification.RecipientUserID)
					continue
				}
			}
		}
		if len(notification.Buttons) > 0 {
			if _, menuErr := s.sender.Send(ctx, domain.OutgoingMessage{UserID: notification.RecipientUserID, Text: "🏠 Главное меню", Buttons: notification.Buttons}); menuErr != nil {
				s.logger.Warn("send refreshed menu", "error", menuErr, "notification_id", notification.ID, "user_id", notification.RecipientUserID)
			}
		}
	}
	return ctx.Err()
}
