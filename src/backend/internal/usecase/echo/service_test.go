package echo_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/echo"
)

type fakeGateway struct {
	messages []domain.Message
	sent     []domain.Message
}

func (f *fakeGateway) GetInfo(context.Context) (domain.BotInfo, error) {
	return domain.BotInfo{ID: 1, Name: "Echo", Username: "echo"}, nil
}

func (f *fakeGateway) Messages(context.Context) <-chan domain.Message {
	updates := make(chan domain.Message, len(f.messages))
	for _, message := range f.messages {
		updates <- message
	}
	close(updates)
	return updates
}

func (f *fakeGateway) SendMessage(_ context.Context, message domain.Message) error {
	f.sent = append(f.sent, message)
	return nil
}

func TestServiceEchoesMessages(t *testing.T) {
	want := domain.Message{ChatID: 42, Text: "hello"}
	gateway := &fakeGateway{messages: []domain.Message{want}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := echo.NewService(gateway, logger).Run(context.Background()); err != nil {
		t.Fatalf("run service: %v", err)
	}

	if len(gateway.sent) != 1 || gateway.sent[0] != want {
		t.Fatalf("sent = %#v, want %#v", gateway.sent, []domain.Message{want})
	}
}
