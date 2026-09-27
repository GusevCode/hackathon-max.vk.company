package control

import (
	"context"
	"log/slog"
	"testing"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type fakeBot struct{ sent []domain.OutgoingMessage }

func (f *fakeBot) GetInfo(context.Context) (domain.BotInfo, error) {
	return domain.BotInfo{Username: "test"}, nil
}
func (f *fakeBot) Events(context.Context) <-chan domain.Event { return make(chan domain.Event) }
func (f *fakeBot) Send(_ context.Context, message domain.OutgoingMessage) error {
	f.sent = append(f.sent, message)
	return nil
}
func (f *fakeBot) AnswerCallback(context.Context, string, string) error { return nil }

func TestInviteAndTaskWorkflow(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	service := NewService(bot, repo, nil, slog.Default())
	ctx := context.Background()
	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 1, UserID: 1, Text: "/invite manager"}); err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) == 0 {
		t.Fatal("expected invite response")
	}
	code := ""
	for candidate := range repo.invites {
		code = candidate
	}
	if code == "" {
		t.Fatal("invite was not stored")
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 2, UserID: 2, Text: "/join " + code}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 1, UserID: 1, Text: "/invite employee"}); err != nil {
		t.Fatal(err)
	}
	code = ""
	for candidate, invite := range repo.invites {
		if invite.Roles[0] == domain.RoleEmployee {
			code = candidate
		}
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 3, UserID: 3, Text: "/join " + code}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 1, UserID: 1, Text: "/assign 3 2"}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 2, UserID: 2, Text: "/task Уборка лифта | Помыть кабину | 3 | 2030-01-02"}); err != nil {
		t.Fatal(err)
	}
	if len(repo.tasks) != 1 {
		t.Fatalf("expected one task, got %d", len(repo.tasks))
	}
	for id := range repo.tasks {
		if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 3, UserID: 3, Text: "/take " + id}); err != nil {
			t.Fatal(err)
		}
		if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 3, UserID: 3, Text: "/submit " + id + " | Готово", Photos: []domain.Photo{{URL: "https://example.com/photo.jpg"}}}); err != nil {
			t.Fatal(err)
		}
		if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 2, UserID: 2, Text: "/rework " + id + " | Повторить уборку"}); err != nil {
			t.Fatal(err)
		}
		if task, ok := repo.Task(id); !ok || task.Status != domain.TaskRework {
			t.Fatalf("expected rework status, got %#v", task)
		}
	}
}
