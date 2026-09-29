package maxbot

import (
	"testing"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

func TestNormalizeUpdateBotStarted(t *testing.T) {
	update := &schemes.BotStartedUpdate{
		ChatId:  42,
		User:    schemes.User{UserId: 42, FirstName: "Иван", LastName: "Петров"},
		Payload: "invite-code",
	}

	event, ok := normalizeUpdate(update)
	if !ok {
		t.Fatal("normalizeUpdate() did not recognize bot_started")
	}
	if event.Kind != domain.EventStarted || event.ChatID != 42 || event.UserID != 42 {
		t.Fatalf("unexpected event: %#v", event)
	}
	if event.DisplayName != "Иван Петров" || event.Payload != "invite-code" {
		t.Fatalf("user data was not preserved: %#v", event)
	}
}
