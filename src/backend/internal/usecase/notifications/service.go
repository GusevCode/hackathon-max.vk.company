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
		err := s.sender.Send(ctx, domain.OutgoingMessage{
			UserID:  notification.RecipientUserID,
			Text:    notification.Text,
			Buttons: notification.Buttons,
		})
		if err != nil {
			s.logger.Warn("send notification", "error", err, "notification_id", notification.ID, "kind", notification.Kind, "user_id", notification.RecipientUserID)
		}
	}
	return ctx.Err()
}
