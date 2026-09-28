package notifications

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type fakeSubscriber struct {
	events chan domain.Notification
}

func (f *fakeSubscriber) Subscribe(context.Context) (<-chan domain.Notification, error) {
	return f.events, nil
}

type fakeSender struct {
	messages chan domain.OutgoingMessage
}

func (f *fakeSender) Send(_ context.Context, message domain.OutgoingMessage) (string, error) {
	f.messages <- message
	return "message-1", nil
}

func TestRunDeliversNotificationToUser(t *testing.T) {
	subscriber := &fakeSubscriber{events: make(chan domain.Notification, 1)}
	sender := &fakeSender{messages: make(chan domain.OutgoingMessage, 1)}
	service := NewService(subscriber, sender, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- service.Run(ctx) }()
	subscriber.events <- domain.Notification{ID: "N1", RecipientUserID: 42, Text: "Новое задание", Buttons: []domain.Button{{Text: "Открыть", Payload: "task:view:T1"}}, Photos: []domain.Photo{{URL: "https://cdn.max.ru/photo.jpg"}}}

	select {
	case message := <-sender.messages:
		if message.UserID != 42 || message.Text != "Новое задание" || len(message.Buttons) != 0 || len(message.Photos) != 1 {
			t.Fatalf("unexpected outgoing message: %#v", message)
		}
	case <-time.After(time.Second):
		t.Fatal("notification was not delivered")
	}
	select {
	case message := <-sender.messages:
		if message.Text != "🏠 Главное меню" || len(message.Buttons) == 0 {
			t.Fatalf("menu was not refreshed separately: %#v", message)
		}
	case <-time.After(time.Second):
		t.Fatal("menu was not refreshed")
	}
	cancel()
}
