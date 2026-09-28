package control

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type fakeBot struct{ sent []domain.OutgoingMessage }

func (f *fakeBot) GetInfo(context.Context) (domain.BotInfo, error) {
	return domain.BotInfo{Username: "control_bot"}, nil
}
func (f *fakeBot) Events(context.Context) <-chan domain.Event { return make(chan domain.Event) }
func (f *fakeBot) Send(_ context.Context, message domain.OutgoingMessage) error {
	f.sent = append(f.sent, message)
	return nil
}
func (f *fakeBot) AnswerCallback(context.Context, string, string) error { return nil }

func TestMenuInvitesAllOrganizationRoles(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	service := NewService(bot, repo, nil, slog.Default())
	ctx := context.Background()

	callback := func(userID int64, payload string) {
		t.Helper()
		if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: userID, UserID: userID, MessageID: "menu-message", Payload: payload}); err != nil {
			t.Fatal(err)
		}
	}
	callback(1, "menu:admin")
	callback(1, "admin:invite")
	callback(1, "admin:invite:operator")
	callback(1, "admin:invite")
	callback(1, "admin:invite:manager")
	callback(1, "admin:invite")
	callback(1, "admin:invite:employee")

	wantRoles := map[domain.Role]bool{}
	for _, invite := range repo.invites {
		if len(invite.Roles) == 1 {
			wantRoles[invite.Roles[0]] = true
		}
	}
	for _, role := range []domain.Role{domain.RoleOperator, domain.RoleManager, domain.RoleEmployee} {
		if !wantRoles[role] {
			t.Fatalf("role %s was not invited; invites = %#v", role, repo.invites)
		}
	}
	if len(bot.sent) < 6 {
		t.Fatalf("sent messages = %d, want menu-driven responses", len(bot.sent))
	}
	if bot.sent[len(bot.sent)-1].MessageID != "menu-message" {
		t.Fatalf("last callback did not edit existing menu message: %#v", bot.sent[len(bot.sent)-1])
	}
}

func TestInviteCodeRegistrationUsesMenu(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	service := NewService(bot, repo, nil, slog.Default())
	ctx := context.Background()

	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 42, UserID: 42, Text: "Привет"}); err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 1 || bot.sent[0].Buttons[0].Payload != "auth:invite" {
		t.Fatalf("unregistered user did not receive registration menu: %#v", bot.sent)
	}
	invite := domain.Invite{Code: "JOIN1234", OrganizationID: "system", Roles: []domain.Role{domain.RoleEmployee}, ExpiresAt: time.Now().Add(time.Hour)}
	repo.SaveInvite(invite)
	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 42, UserID: 42, Payload: "auth:invite"}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 42, UserID: 42, Text: "join1234"}); err != nil {
		t.Fatal(err)
	}
	user, ok := repo.UserByMaxID(42)
	if !ok || !user.HasRole(domain.RoleEmployee) {
		t.Fatalf("user was not registered from invite: %#v, %v", user, ok)
	}
}

func TestCommandsDoNotMutateState(t *testing.T) {
	service := NewService(&fakeBot{}, NewMemoryRepository(1), nil, slog.Default())
	if err := service.handle(context.Background(), domain.Event{Kind: domain.EventMessage, ChatID: 1, UserID: 1, Text: "/invite employee"}); err != nil {
		t.Fatal(err)
	}
	if len(service.repo.(*MemoryRepository).invites) != 0 {
		t.Fatal("manual command unexpectedly created an invite")
	}
}

func TestEmployeeTaskActionsAreMenuDriven(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	repo.SaveUser(domain.User{ID: "employee", OrganizationID: "system", MaxUserID: 2, DisplayName: "Сотрудник", Roles: []domain.Role{domain.RoleEmployee}, Status: domain.UserActive})
	task := domain.Task{ID: "TASK01", OrganizationID: "system", Title: "Уборка лифта", Description: "Помыть кабину", AssigneeID: "employee", ManagerID: "initial-admin", Status: domain.TaskAssigned, DueAt: time.Now().Add(time.Hour)}
	repo.SaveTask(task)
	service := NewService(bot, repo, nil, slog.Default())
	ctx := context.Background()

	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 2, UserID: 2, MessageID: "task-menu", Payload: "task:view:TASK01"}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 2, UserID: 2, MessageID: "task-menu", Payload: "task:take:TASK01"}); err != nil {
		t.Fatal(err)
	}
	updated, _ := repo.Task("TASK01")
	if updated.Status != domain.TaskInProgress {
		t.Fatalf("task status = %s, want in progress", updated.Status)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 2, UserID: 2, MessageID: "task-menu", Payload: "task:submit:TASK01"}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 2, UserID: 2, Photos: []domain.Photo{{URL: "https://example.com/photo.jpg"}}}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 2, UserID: 2, Text: "Кабина и кнопки очищены"}); err != nil {
		t.Fatal(err)
	}
	updated, _ = repo.Task("TASK01")
	if updated.Status != domain.TaskSubmitted {
		t.Fatalf("task status = %s, want submitted", updated.Status)
	}
}

func TestCallbackRendersExistingMessage(t *testing.T) {
	bot := &fakeBot{}
	service := NewService(bot, NewMemoryRepository(1), nil, slog.Default())

	err := service.handle(context.Background(), domain.Event{Kind: domain.EventCallback, ChatID: 1, UserID: 1, MessageID: "message-123", Payload: "menu:help"})
	if err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 1 || bot.sent[0].MessageID != "message-123" {
		t.Fatalf("callback should edit existing message: %#v", bot.sent)
	}
}

func TestTextStepRendersIntoLastMenuMessage(t *testing.T) {
	bot := &fakeBot{}
	service := NewService(bot, NewMemoryRepository(1), nil, slog.Default())
	ctx := context.Background()

	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 1, UserID: 1, MessageID: "menu-1", Payload: "menu:help"}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 1, UserID: 1, Text: "текстовый шаг"}); err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 2 || bot.sent[1].MessageID != "menu-1" {
		t.Fatalf("text step should edit the last menu message: %#v", bot.sent)
	}
}

func TestRegistrationStoresDisplayName(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	service := NewService(bot, repo, nil, slog.Default())
	ctx := context.Background()
	repo.SaveInvite(domain.Invite{Code: "JOIN1234", OrganizationID: "system", Roles: []domain.Role{domain.RoleEmployee}, ExpiresAt: time.Now().Add(time.Hour)})

	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 42, UserID: 42, MessageID: "registration", Payload: "auth:invite", DisplayName: "Иван Петров"}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 42, UserID: 42, Text: "join1234", DisplayName: "Иван Петров"}); err != nil {
		t.Fatal(err)
	}
	user, ok := repo.UserByMaxID(42)
	if !ok || user.DisplayName != "Иван Петров" {
		t.Fatalf("display name = %q, want MAX name", user.DisplayName)
	}
}
