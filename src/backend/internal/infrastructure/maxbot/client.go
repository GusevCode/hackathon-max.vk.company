package maxbot

import (
	"context"
	"fmt"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	maxapi "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

type Client struct {
	api *maxapi.Api
}

func New(token string) (*Client, error) {
	api, err := maxapi.New(token)
	if err != nil {
		return nil, fmt.Errorf("create MAX client: %w", err)
	}
	return &Client{api: api}, nil
}

func (c *Client) GetInfo(ctx context.Context) (domain.BotInfo, error) {
	info, err := c.api.Bots.GetBot(ctx)
	if err != nil {
		return domain.BotInfo{}, err
	}
	return domain.BotInfo{ID: info.UserId, Name: info.Name, Username: info.Username}, nil
}

func (c *Client) Messages(ctx context.Context) <-chan domain.Message {
	messages := make(chan domain.Message)
	go func() {
		defer close(messages)
		for update := range c.api.GetUpdates(ctx) {
			messageUpdate, ok := update.(*schemes.MessageCreatedUpdate)
			if !ok || messageUpdate.GetText() == "" {
				continue
			}
			select {
			case messages <- domain.Message{ChatID: messageUpdate.GetChatID(), Text: messageUpdate.GetText()}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return messages
}

func (c *Client) SendMessage(ctx context.Context, message domain.Message) error {
	return c.api.Messages.Send(ctx, maxapi.NewMessage().SetChat(message.ChatID).SetText(message.Text))
}
