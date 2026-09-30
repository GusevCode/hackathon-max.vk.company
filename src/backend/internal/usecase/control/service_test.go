package control

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type fakeBot struct {
	sent    []domain.OutgoingMessage
	deleted []string
}

type fakePhotoStore struct{}

func (fakePhotoStore) UploadURL(context.Context, string, string, string) error { return nil }
func (fakePhotoStore) Read(context.Context, string, int64) ([]byte, string, error) {
	return []byte("fake-image"), "image/jpeg", nil
}
func (fakePhotoStore) DeleteObject(context.Context, string) error { return nil }
func (fakePhotoStore) DeleteAllTaskPhotos(context.Context) error  { return nil }

type fakeInspectionPublisher struct {
	requests []domain.InspectionRequested
}

func (f *fakeInspectionPublisher) PublishInspection(_ context.Context, request domain.InspectionRequested) error {
	f.requests = append(f.requests, request)
	return nil
}

func (f *fakeBot) GetInfo(context.Context) (domain.BotInfo, error) {
	return domain.BotInfo{Username: "control_bot"}, nil
}
func (f *fakeBot) Events(context.Context) <-chan domain.Event { return make(chan domain.Event) }
func (f *fakeBot) Send(_ context.Context, message domain.OutgoingMessage) (string, error) {
	f.sent = append(f.sent, message)
	if message.MessageID != "" {
		return message.MessageID, nil
	}
	return fmt.Sprintf("sent-%d", len(f.sent)), nil
}
func (f *fakeBot) DeleteMessage(_ context.Context, messageID string) error {
	f.deleted = append(f.deleted, messageID)
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

func TestBotStartedOpensRegistrationMenuWithoutMessage(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	service := NewService(bot, repo, nil, slog.Default())

	if err := service.handle(context.Background(), domain.Event{Kind: domain.EventStarted, ChatID: 42, UserID: 42, DisplayName: "Иван Петров"}); err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 1 || bot.sent[0].ChatID != 42 || bot.sent[0].Buttons[0].Payload != "auth:invite" {
		t.Fatalf("start event did not open registration menu: %#v", bot.sent)
	}
}

func TestBotStartedOpensMainMenuForRegisteredUser(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	repo.SaveUser(domain.User{ID: "user-42", OrganizationID: "system", MaxUserID: 42, DisplayName: "Иван Петров", Roles: []domain.Role{domain.RoleEmployee}, Status: domain.UserActive})
	service := NewService(bot, repo, nil, slog.Default())

	if err := service.handle(context.Background(), domain.Event{Kind: domain.EventStarted, ChatID: 42, UserID: 42, DisplayName: "Иван Петров"}); err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 1 || bot.sent[0].ChatID != 42 || len(bot.sent[0].Buttons) == 0 {
		t.Fatalf("start event did not open main menu: %#v", bot.sent)
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

func TestSubmittingPhotosRequestsPolzaAnalysis(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	employee := domain.User{ID: "employee", OrganizationID: "system", MaxUserID: 2, Roles: []domain.Role{domain.RoleEmployee}, Status: domain.UserActive}
	repo.SaveUser(employee)
	repo.SaveTask(domain.Task{
		ID: "TASKAI", OrganizationID: "system", Title: "Elevator cleaning", Description: "Clean floor and buttons",
		AssigneeID: employee.ID, ManagerID: "initial-admin", Status: domain.TaskInProgress,
	})
	publisher := &fakeInspectionPublisher{}
	service := NewService(bot, repo, fakePhotoStore{}, slog.Default())
	service.SetInspectionPublisher(publisher, "v1")
	err := service.submitPhotos(context.Background(), domain.Event{ChatID: 2, UserID: 2}, employee, mustTask(t, repo, "TASKAI"), []domain.Photo{{URL: "https://cdn.max.ru/after.jpg"}}, "Cleaning completed")
	if err != nil {
		t.Fatal(err)
	}
	if len(publisher.requests) != 1 {
		t.Fatalf("inspection requests = %d, want 1", len(publisher.requests))
	}
	request := publisher.requests[0]
	if request.TaskID != "TASKAI" || request.SubmissionID == "" || len(request.Images) != 1 || request.Images[0].Kind != "after" {
		t.Fatalf("unexpected inspection request: %#v", request)
	}
	analysis, ok := repo.Analysis("TASKAI")
	if !ok || analysis.Status != domain.AnalysisPending || analysis.SubmissionID != request.SubmissionID {
		t.Fatalf("pending analysis was not saved: %#v", analysis)
	}
}

func TestSubmittingMultiplePhotosRequestsOnlyLatestPhotoForAnalysis(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	employee := domain.User{ID: "employee", OrganizationID: "system", MaxUserID: 2, Roles: []domain.Role{domain.RoleEmployee}, Status: domain.UserActive}
	repo.SaveUser(employee)
	repo.SaveEvidence(domain.Evidence{ID: "before-old", TaskID: "TASKLATEST", Kind: "before", ObjectKey: "before-old.jpg", CreatedAt: time.Unix(100, 0)})
	repo.SaveEvidence(domain.Evidence{ID: "before-latest", TaskID: "TASKLATEST", Kind: "before", ObjectKey: "before-latest.jpg", CreatedAt: time.Unix(200, 0)})
	repo.SaveTask(domain.Task{
		ID: "TASKLATEST", OrganizationID: "system", Title: "Уборка лифта", Description: "Помыть пол и кнопки",
		AssigneeID: employee.ID, ManagerID: "initial-admin", Status: domain.TaskInProgress,
	})
	publisher := &fakeInspectionPublisher{}
	service := NewService(bot, repo, fakePhotoStore{}, slog.Default())
	service.SetInspectionPublisher(publisher, "v1")

	err := service.submitPhotos(context.Background(), domain.Event{ChatID: 2, UserID: 2}, employee, mustTask(t, repo, "TASKLATEST"), []domain.Photo{
		{URL: "https://cdn.max.ru/after-old.jpg"},
		{URL: "https://cdn.max.ru/after-latest.jpg"},
	}, "Работа выполнена")
	if err != nil {
		t.Fatal(err)
	}
	if len(publisher.requests) != 1 {
		t.Fatalf("inspection requests = %d, want 1", len(publisher.requests))
	}
	request := publisher.requests[0]
	if len(request.Images) != 2 {
		t.Fatalf("inspection images = %d, want latest before and latest after", len(request.Images))
	}
	if request.Images[0].Kind != "before" || request.Images[0].ObjectKey != "before-latest.jpg" {
		t.Fatalf("unexpected before image: %#v", request.Images[0])
	}
	afterEvidence := repo.Evidences("TASKLATEST")
	latestAfter := afterEvidence[len(afterEvidence)-1]
	if request.Images[1].Kind != "after" || request.Images[1].ObjectKey != latestAfter.ObjectKey {
		t.Fatalf("unexpected after image: %#v, latest evidence: %#v", request.Images[1], latestAfter)
	}
}

func TestManagerTaskCardShowsAnalysisAsRecommendation(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	task := domain.Task{
		ID: "TASKCARD", OrganizationID: "system", Title: "Elevator cleaning",
		ManagerID: "initial-admin", Status: domain.TaskSubmitted, SubmissionID: "submission-1",
	}
	repo.SaveTask(task)
	repo.SaveAnalysis(domain.EvidenceAnalysis{
		ID: "analysis-1", TaskID: task.ID, SubmissionID: task.SubmissionID,
		Status: domain.AnalysisSucceeded, Recommendation: domain.RecommendationApprove,
		Confidence: 0.82, Observations: []string{"floor appears clean"},
	})
	service := NewService(bot, repo, nil, slog.Default())
	manager, _ := repo.UserByMaxID(1)
	if err := service.taskCard(context.Background(), domain.Event{ChatID: 1}, manager, task); err != nil {
		t.Fatal(err)
	}
	message := bot.sent[len(bot.sent)-1].Text
	if !strings.Contains(message, "🤖 Предварительный ИИ-анализ") || !strings.Contains(message, "ИИ не принимает решение") {
		t.Fatalf("analysis disclaimer is missing from card: %q", message)
	}
	if count := strings.Count(message, "🤖 Предварительный ИИ-анализ"); count != 1 {
		t.Fatalf("analysis section appears %d times, want exactly once: %q", count, message)
	}
}

func mustTask(t *testing.T, repo *MemoryRepository, id string) domain.Task {
	t.Helper()
	task, ok := repo.Task(id)
	if !ok {
		t.Fatalf("task %q not found", id)
	}
	return task
}

func TestManagerCanViewBeforePhotos(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	task := domain.Task{
		ID:             "TASKBEFORE",
		OrganizationID: "system",
		Title:          "Уборка лифта",
		Description:    "Помыть кабину",
		ManagerID:      "initial-admin",
		Status:         domain.TaskInProgress,
		BeforePhotos:   []domain.Photo{{URL: "https://cdn.max.ru/before.jpg"}},
	}
	repo.SaveTask(task)
	service := NewService(bot, repo, nil, slog.Default())

	err := service.handle(context.Background(), domain.Event{
		Kind:      domain.EventCallback,
		ChatID:    1,
		UserID:    1,
		MessageID: "task-menu",
		Payload:   "task:before_photos:TASKBEFORE",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 1 || len(bot.sent[0].Photos) != 1 {
		t.Fatalf("before photos were not sent: %#v", bot.sent)
	}
}

func TestManagerCanDeleteBeforePhoto(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	repo.SaveTask(domain.Task{
		ID:             "TASKPHOTO",
		OrganizationID: "system",
		Title:          "Уборка лифта",
		ManagerID:      "initial-admin",
		Status:         domain.TaskInProgress,
		BeforePhotos:   []domain.Photo{{URL: "https://cdn.max.ru/before.jpg"}},
	})
	service := NewService(bot, repo, nil, slog.Default())
	ctx := context.Background()

	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 1, UserID: 1, MessageID: "photos-menu", Payload: "task:before_photos:TASKPHOTO"}); err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 1 || len(bot.sent[0].Buttons) < 2 {
		t.Fatalf("photo menu does not contain delete action: %#v", bot.sent)
	}

	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 1, UserID: 1, MessageID: "photos-menu", Payload: "task:delete_photo:TASKPHOTO:before:0"}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 1, UserID: 1, MessageID: "photos-menu", Payload: "task:delete_photo_confirm:TASKPHOTO:before:0"}); err != nil {
		t.Fatal(err)
	}
	updated, ok := repo.Task("TASKPHOTO")
	if !ok || len(updated.BeforePhotos) != 0 {
		t.Fatalf("before photo was not deleted: %#v", updated.BeforePhotos)
	}
}

func TestManagerCanSendTaskToRework(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	repo.SaveUser(domain.User{ID: "employee", OrganizationID: "system", MaxUserID: 2, DisplayName: "Сотрудник", Roles: []domain.Role{domain.RoleEmployee}, Status: domain.UserActive})
	repo.SaveTask(domain.Task{
		ID:             "TASKREWORK",
		OrganizationID: "system",
		Title:          "Уборка лифта",
		Description:    "Помыть кабину",
		AssigneeID:     "employee",
		ManagerID:      "initial-admin",
		Status:         domain.TaskSubmitted,
	})
	service := NewService(bot, repo, nil, slog.Default())
	ctx := context.Background()

	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 1, UserID: 1, MessageID: "task-menu", Payload: "task:rework:TASKREWORK"}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 1, UserID: 1, Text: "Нужно домыть углы"}); err != nil {
		t.Fatal(err)
	}
	updated, ok := repo.Task("TASKREWORK")
	if !ok || updated.Status != domain.TaskRework || updated.Comment != "Нужно домыть углы" {
		t.Fatalf("task was not sent to rework: %#v", updated)
	}
}

func TestAdminCanClearTasksWithoutRemovingEmployees(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	repo.SaveUser(domain.User{ID: "employee", OrganizationID: "system", MaxUserID: 2, DisplayName: "Сотрудник", Roles: []domain.Role{domain.RoleEmployee}, Status: domain.UserActive})
	repo.SaveTask(domain.Task{ID: "TASKCLEAR", OrganizationID: "system", Title: "Тест", ManagerID: "initial-admin"})
	service := NewService(bot, repo, nil, slog.Default())
	ctx := context.Background()

	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 1, UserID: 1, MessageID: "admin-menu", Payload: "admin:clear"}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 1, UserID: 1, MessageID: "admin-menu", Payload: "admin:clear:yes"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.Task("TASKCLEAR"); ok {
		t.Fatal("task was not removed")
	}
	if _, ok := repo.UserByMaxID(2); !ok {
		t.Fatal("employee was removed during cleanup")
	}
}

func TestManagerCanCloseTaskAndHideItFromActiveLists(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	manager := domain.User{ID: "manager", OrganizationID: "system", MaxUserID: 3, DisplayName: "Руководитель", Roles: []domain.Role{domain.RoleManager}, Status: domain.UserActive}
	employee := domain.User{ID: "employee", OrganizationID: "system", MaxUserID: 2, DisplayName: "Сотрудник", Roles: []domain.Role{domain.RoleEmployee}, Status: domain.UserActive, ManagerID: manager.ID}
	repo.SaveUser(manager)
	repo.SaveUser(employee)
	repo.SaveTask(domain.Task{
		ID:             "TASKCLOSE",
		OrganizationID: "system",
		Title:          "Уборка лифта",
		AssigneeID:     employee.ID,
		ManagerID:      manager.ID,
		Status:         domain.TaskInProgress,
	})
	service := NewService(bot, repo, nil, slog.Default())
	ctx := context.Background()

	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: manager.MaxUserID, UserID: manager.MaxUserID, MessageID: "task-menu", Payload: "task:close:TASKCLOSE"}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: manager.MaxUserID, UserID: manager.MaxUserID, MessageID: "task-menu", Payload: "task:close_confirm:TASKCLOSE"}); err != nil {
		t.Fatal(err)
	}

	closed, ok := repo.Task("TASKCLOSE")
	if !ok || closed.Status != domain.TaskClosed {
		t.Fatalf("task was not closed: %#v", closed)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: employee.MaxUserID, UserID: employee.MaxUserID, MessageID: "employee-menu", Payload: "task:view:TASKCLOSE"}); err != nil {
		t.Fatal(err)
	}
	last := bot.sent[len(bot.sent)-1]
	if last.Text != "Задание не найдено или недоступно.\n\n🏠 Главное меню" {
		t.Fatalf("closed task remains accessible to employee: %#v", last)
	}
}

func TestAdminCanRemoveRoleButNotLastRole(t *testing.T) {
	bot := &fakeBot{}
	repo := NewMemoryRepository(1)
	target := domain.User{
		ID:             "multi-role-user",
		OrganizationID: "system",
		MaxUserID:      42,
		DisplayName:    "Иван",
		Roles:          []domain.Role{domain.RoleEmployee, domain.RoleManager},
		Status:         domain.UserActive,
	}
	repo.SaveUser(target)
	service := NewService(bot, repo, nil, slog.Default())
	ctx := context.Background()

	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 1, UserID: 1, MessageID: "roles-menu", Payload: "user:role_remove:42:manager"}); err != nil {
		t.Fatal(err)
	}
	updated, _ := repo.UserByMaxID(42)
	if updated.HasRole(domain.RoleManager) || !updated.HasRole(domain.RoleEmployee) {
		t.Fatalf("manager role was not removed correctly: %#v", updated.Roles)
	}

	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 1, UserID: 1, MessageID: "roles-menu", Payload: "user:role_remove:42:employee"}); err != nil {
		t.Fatal(err)
	}
	updated, _ = repo.UserByMaxID(42)
	if !updated.HasRole(domain.RoleEmployee) || len(updated.Roles) != 1 {
		t.Fatalf("last role should not be removed: %#v", updated.Roles)
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

func TestTextStepReplacesLastMenuMessage(t *testing.T) {
	bot := &fakeBot{}
	service := NewService(bot, NewMemoryRepository(1), nil, slog.Default())
	ctx := context.Background()

	if err := service.handle(ctx, domain.Event{Kind: domain.EventCallback, ChatID: 1, UserID: 1, MessageID: "menu-1", Payload: "menu:help"}); err != nil {
		t.Fatal(err)
	}
	if err := service.handle(ctx, domain.Event{Kind: domain.EventMessage, ChatID: 1, UserID: 1, Text: "текстовый шаг"}); err != nil {
		t.Fatal(err)
	}
	if len(bot.sent) != 2 || bot.sent[1].MessageID != "" {
		t.Fatalf("text step should send a new menu message: %#v", bot.sent)
	}
	if len(bot.deleted) != 1 || bot.deleted[0] != "menu-1" {
		t.Fatalf("text step should delete the previous menu message: %#v", bot.deleted)
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
