package notifications

import (
	"context"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type Publisher interface {
	Publish(context.Context, domain.Notification) error
}

type Subscriber interface {
	Subscribe(context.Context) (<-chan domain.Notification, error)
}

type Sender interface {
	Send(context.Context, domain.OutgoingMessage) (string, error)
}
