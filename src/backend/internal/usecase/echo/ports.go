package echo

import (
	"context"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

// BotGateway hides the MAX SDK from the application layer.
type BotGateway interface {
	GetInfo(ctx context.Context) (domain.BotInfo, error)
	Messages(ctx context.Context) <-chan domain.Message
	SendMessage(ctx context.Context, message domain.Message) error
}
