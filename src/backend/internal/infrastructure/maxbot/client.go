package maxbot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	maxapi "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

type Client struct {
	api     *maxapi.Api
	token   string
	updates chan schemes.UpdateInterface
}

func New(token string) (*Client, error) {
	api, err := maxapi.New(token)
	if err != nil {
		return nil, fmt.Errorf("create MAX client: %w", err)
	}
	return &Client{api: api, token: token, updates: make(chan schemes.UpdateInterface, 100)}, nil
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
		return domain.Event{Kind: domain.EventMessage, ChatID: value.GetChatID(), UserID: value.GetUserID(), DisplayName: maxUserDisplayName(value.Message.Sender), Text: value.GetText(), Photos: photos}, true
	case *schemes.MessageCallbackUpdate:
		messageID := ""
		if value.Message != nil {
			messageID = value.Message.Body.Mid
		}
		return domain.Event{Kind: domain.EventCallback, ChatID: value.GetChatID(), UserID: value.GetUserID(), DisplayName: maxUserDisplayName(value.Callback.User), MessageID: messageID, Payload: value.Callback.Payload, CallbackID: value.Callback.CallbackID}, true
	default:
		return domain.Event{}, false
	}
}

func maxUserDisplayName(user schemes.User) string {
	if user.Name != "" {
		return user.Name
	}
	if user.FirstName != "" || user.LastName != "" {
		return strings.TrimSpace(user.FirstName + " " + user.LastName)
	}
	return user.Username
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

func (c *Client) Send(ctx context.Context, message domain.OutgoingMessage) (string, error) {
	msg := maxapi.NewMessage().SetText(message.Text)
	if message.UserID != 0 {
		msg.SetUser(message.UserID)
	} else {
		msg.SetChat(message.ChatID)
	}
	c.addKeyboard(msg, message.Buttons)
	if message.MessageID != "" {
		return message.MessageID, c.api.Messages.EditMessage(ctx, message.MessageID, msg)
	}
	if err := c.addPhotos(ctx, msg, message.Photos); err != nil {
		return "", err
	}
	result, err := c.api.Messages.SendWithResult(ctx, msg)
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", fmt.Errorf("MAX returned an empty message")
	}
	return result.Body.Mid, nil
}

func (c *Client) addKeyboard(msg *maxapi.Message, buttons []domain.Button) {
	if len(buttons) == 0 {
		return
	}
	keyboard := c.api.Messages.NewKeyboardBuilder()
	rows := make(map[int]*maxapi.KeyboardRow)
	for _, button := range buttons {
		row, ok := rows[button.Row]
		if !ok {
			row = keyboard.AddRow()
			rows[button.Row] = row
		}
		row.AddCallback(button.Text, schemes.DEFAULT, button.Payload)
	}
	msg.AddKeyboard(keyboard)
}

func (c *Client) addPhotos(ctx context.Context, msg *maxapi.Message, photos []domain.Photo) error {
	for _, photo := range photos {
		body, err := c.downloadPhoto(ctx, photo)
		if err != nil {
			return err
		}
		tokens, uploadErr := c.api.Uploads.UploadPhotoFromReader(ctx, body)
		_ = body.Close()
		if uploadErr != nil {
			return fmt.Errorf("upload photo to MAX: %w", uploadErr)
		}
		msg.AddPhoto(tokens)
	}
	return nil
}

func (c *Client) downloadPhoto(ctx context.Context, photo domain.Photo) (io.ReadCloser, error) {
	if strings.TrimSpace(photo.URL) == "" {
		return nil, fmt.Errorf("download photo from MAX: URL is empty")
	}
	parsedURL, err := url.Parse(photo.URL)
	if err != nil {
		return nil, fmt.Errorf("parse photo URL: %w", err)
	}
	if !isTrustedMAXHost(parsedURL.Hostname()) {
		return nil, fmt.Errorf("download photo from MAX: untrusted host %q", parsedURL.Hostname())
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, photo.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create photo request: %w", err)
	}
	if c.token != "" {
		request.Header.Set("Authorization", c.token)
	}
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("download photo from MAX: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_ = response.Body.Close()
		return nil, fmt.Errorf("download photo from MAX: HTTP %s", response.Status)
	}
	return response.Body, nil
}

func isTrustedMAXHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	for _, suffix := range []string{"max.ru", "oneme.ru", "okcdn.ru"} {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

func (c *Client) DeleteMessage(ctx context.Context, messageID string) error {
	if messageID == "" {
		return nil
	}
	_, err := c.api.Messages.DeleteMessage(ctx, messageID)
	return err
}

func (c *Client) AnswerCallback(ctx context.Context, callbackID, notification string) error {
	_, err := c.api.Messages.AnswerOnCallback(ctx, callbackID, &schemes.CallbackAnswer{Notification: notification})
	return err
}
