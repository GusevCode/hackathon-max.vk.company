package maxbot

import (
	"context"
	"fmt"
	"net/http"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	maxapi "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

type Client struct {
	api     *maxapi.Api
	updates chan schemes.UpdateInterface
}

func New(token string) (*Client, error) {
	api, err := maxapi.New(token)
	if err != nil {
		return nil, fmt.Errorf("create MAX client: %w", err)
	}
	return &Client{api: api, updates: make(chan schemes.UpdateInterface, 100)}, nil
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
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-c.updates:
				messageUpdate, ok := update.(*schemes.MessageCreatedUpdate)
				if !ok || messageUpdate.GetText() == "" {
					continue
				}
				select {
				case messages <- domain.Message{ChatID: messageUpdate.GetChatID(), UserID: messageUpdate.GetUserID(), Text: messageUpdate.GetText()}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return messages
}

func (c *Client) Events(ctx context.Context) <-chan domain.Event {
	events := make(chan domain.Event)
	go func() {
		defer close(events)
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-c.updates:
				event, ok := normalizeUpdate(update)
				if !ok {
					continue
				}
				select {
				case events <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return events
}

func normalizeUpdate(update schemes.UpdateInterface) (domain.Event, bool) {
	switch value := update.(type) {
	case *schemes.MessageCreatedUpdate:
		photos := make([]domain.Photo, 0)
		for _, attachment := range value.Message.Body.Attachments {
			if photo, ok := attachment.(*schemes.PhotoAttachment); ok {
				photos = append(photos, domain.Photo{URL: photo.Payload.Url, Token: photo.Payload.Token})
			}
		}
		return domain.Event{Kind: domain.EventMessage, ChatID: value.GetChatID(), UserID: value.GetUserID(), Text: value.GetText(), Photos: photos}, true
	case *schemes.MessageCallbackUpdate:
		messageID := ""
		if value.Message != nil {
			messageID = value.Message.Body.Mid
		}
		return domain.Event{Kind: domain.EventCallback, ChatID: value.GetChatID(), UserID: value.GetUserID(), MessageID: messageID, Payload: value.Callback.Payload, CallbackID: value.Callback.CallbackID}, true
	default:
		return domain.Event{}, false
	}
}

func (c *Client) WebhookHandler(secret string) http.Handler {
	return c.api.GetUpdateHandler(c.updates, secret)
}

func (c *Client) Subscribe(ctx context.Context, webhookURL, secret string) error {
	_, err := c.api.Subscriptions.Subscribe(ctx, webhookURL, []string{}, secret)
	return err
}

func (c *Client) SendMessage(ctx context.Context, message domain.Message) error {
	return c.api.Messages.Send(ctx, maxapi.NewMessage().SetChat(message.ChatID).SetText(message.Text))
}

func (c *Client) Send(ctx context.Context, message domain.OutgoingMessage) error {
	msg := maxapi.NewMessage().SetChat(message.ChatID).SetText(message.Text)
	if len(message.Buttons) > 0 {
		keyboard := c.api.Messages.NewKeyboardBuilder()
		rows := make(map[int]*maxapi.KeyboardRow)
		for _, button := range message.Buttons {
			row, ok := rows[button.Row]
			if !ok {
				row = keyboard.AddRow()
				rows[button.Row] = row
			}
			row.AddCallback(button.Text, schemes.DEFAULT, button.Payload)
		}
		msg.AddKeyboard(keyboard)
	}
	if message.MessageID != "" {
		return c.api.Messages.EditMessage(ctx, message.MessageID, msg)
	}
	return c.api.Messages.Send(ctx, msg)
}

func (c *Client) AnswerCallback(ctx context.Context, callbackID, notification string) error {
	_, err := c.api.Messages.AnswerOnCallback(ctx, callbackID, &schemes.CallbackAnswer{Notification: notification})
	return err
}
