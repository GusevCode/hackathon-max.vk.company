package control

import (
	"context"
	"strings"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/access"
)

func (s *Service) handleCallback(ctx context.Context, event domain.Event) error {
	parts := strings.Split(event.Payload, ":")
	user, registered := s.repo.UserByMaxID(event.UserID)
	if parts[0] == "auth" && len(parts) == 2 && parts[1] == "invite" {
		s.setSession(event.UserID, Session{Kind: SessionInviteCode})
		return s.send(ctx, event.ChatID, "🎟 Введите пригласительный код одним сообщением:", nil)
	}
	if !registered || !user.IsActive() {
		return s.showRegistration(ctx, event.ChatID)
	}

	switch {
	case len(parts) == 2 && parts[0] == "menu":
		return s.handleMenuCallback(ctx, event, user, parts[1])
	case len(parts) == 2 && parts[0] == "reference":
		return s.handleReferenceCallback(ctx, event, user, parts[1])
	case len(parts) == 3 && parts[0] == "reference":
		return s.handleReferenceCallback(ctx, event, user, parts[1], parts[2])
	case len(parts) == 2 && parts[0] == "admin":
		return s.handleAdminCallback(ctx, event, user, parts[1])
	case len(parts) == 3 && parts[0] == "admin" && parts[1] == "clear":
		return s.handleClearCallback(ctx, event, user, parts[2])
	case len(parts) == 3 && parts[0] == "admin" && parts[1] == "invite":
		return s.createInvite(ctx, event, user, domain.Role(parts[2]))
	case len(parts) == 3 && parts[0] == "user":
		return s.handleUserCallback(ctx, event, user, parts[1], parts[2])
	case len(parts) == 4 && parts[0] == "user" && parts[1] == "manager":
		return s.assignManager(ctx, event, user, parts[2], parts[3])
	case len(parts) == 4 && parts[0] == "user" && parts[1] == "role":
		return s.addUserRole(ctx, event, user, parts[2], domain.Role(parts[3]))
	case len(parts) == 3 && parts[0] == "task":
		return s.handleTaskCallback(ctx, event, user, parts[1], parts[2])
	case len(parts) == 3 && parts[0] == "wizard":
		return s.handleWizardCallback(ctx, event, user, parts[1], parts[2])
	default:
		return s.sendHome(ctx, event.ChatID, user, "Действие недоступно.")
	}
}

func (s *Service) handleMenuCallback(ctx context.Context, event domain.Event, user domain.User, target string) error {
	s.clearSession(event.UserID)
	switch target {
	case "home":
		return s.sendHome(ctx, event.ChatID, user, "Готово.")
	case "help":
		return s.send(ctx, event.ChatID, helpText(), append(menuForUser(user), domain.Button{Text: "↩️ Главное меню", Payload: "menu:home", Row: 99}))
	case "tasks":
		return s.listTasks(ctx, event, user)
	case "create_task":
		if !access.Can(user, access.ActionCreateTask) {
			return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
		}
		s.setSession(event.UserID, Session{Kind: SessionTaskTitle})
		return s.send(ctx, event.ChatID, "➕ НОВОЕ ЗАДАНИЕ\n\nНапишите название работы:", nil)
	case "users":
		return s.listUsers(ctx, event, user)
	case "objects":
		return s.listObjects(ctx, event, user)
	case "worktypes":
		return s.listWorkTypes(ctx, event, user)
	case "admin":
		if !user.HasRole(domain.RoleAdmin) && !user.HasRole(domain.RoleOperator) {
			return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
		}
		return s.adminMenu(ctx, event.ChatID, user)
	default:
		return s.sendHome(ctx, event.ChatID, user, "Раздел пока недоступен.")
	}
}

func (s *Service) handleAdminCallback(ctx context.Context, event domain.Event, user domain.User, target string) error {
	if target == "invite" {
		if !access.Can(user, access.ActionViewUsers) {
			return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
		}
		return s.inviteMenu(ctx, event.ChatID, user)
	}
	if target == "users" {
		return s.listUsers(ctx, event, user)
	}
	if target == "clear" {
		return s.handleClearCallback(ctx, event, user, "confirm")
	}
	return s.sendHome(ctx, event.ChatID, user, "Раздел управления не найден.")
}

func (s *Service) handleWizardCallback(ctx context.Context, event domain.Event, user domain.User, kind, value string) error {
	session, ok := s.session(event.UserID)
	if !ok {
		return s.sendHome(ctx, event.ChatID, user, "Сценарий устарел. Начните заново.")
	}
	switch kind {
	case "employee":
		if session.Kind != SessionTaskDueDate {
			return s.sendHome(ctx, event.ChatID, user, "Сценарий создания задания устарел.")
		}
		session.Draft.AssigneeID = value
		session.Kind = SessionTaskDueDate
		s.setSession(event.UserID, session)
		return s.send(ctx, event.ChatID, "📅 Введите срок выполнения в формате ДД.ММ.ГГГГ:", nil)
	case "object":
		session.Draft.ObjectID = value
		s.setSession(event.UserID, session)
		return s.chooseWorkType(ctx, event, user, session)
	case "worktype":
		session.Draft.WorkTypeID = value
		s.setSession(event.UserID, session)
		return s.confirmTask(ctx, event, user, session)
	case "confirm":
		if value == "yes" {
			return s.createTaskFromDraft(ctx, event, user, session)
		}
		s.clearSession(event.UserID)
		return s.sendHome(ctx, event.ChatID, user, "Создание задания отменено.")
	default:
		return s.sendHome(ctx, event.ChatID, user, "Неизвестный шаг сценария.")
	}
}

func (s *Service) handleTaskCallback(ctx context.Context, event domain.Event, user domain.User, action, taskID string) error {
	task, ok := s.repo.Task(taskID)
	if !ok || !access.CanViewTask(user, task) {
		return s.sendHome(ctx, event.ChatID, user, "Задание не найдено или недоступно.")
	}
	switch action {
	case "view":
		return s.taskCard(ctx, event, user, task)
	case "edit":
		return s.taskEditMenu(ctx, event, user, task)
	case "edit_title":
		if !access.CanEditTask(user, task) {
			return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
		}
		s.setSession(event.UserID, Session{Kind: SessionTaskEditTitle, TaskID: task.ID})
		return s.send(ctx, event.ChatID, "📝 Введите новое название задания:", nil)
	case "edit_description":
		if !access.CanEditTask(user, task) {
			return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
		}
		s.setSession(event.UserID, Session{Kind: SessionTaskEditDesc, TaskID: task.ID})
		return s.send(ctx, event.ChatID, "📄 Введите новое описание задания:", nil)
	case "edit_due":
		if !access.CanEditTask(user, task) {
			return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
		}
		s.setSession(event.UserID, Session{Kind: SessionTaskEditDueDate, TaskID: task.ID})
		return s.send(ctx, event.ChatID, "📅 Введите новый срок в формате ДД.ММ.ГГГГ:", nil)
	case "take":
		return s.takeTask(ctx, event.ChatID, user, task)
	case "before":
		s.setSession(event.UserID, Session{Kind: SessionPhotoBefore, TaskID: task.ID})
		return s.send(ctx, event.ChatID, "📷 Прикрепите фотографию ДО выполнения работы:", nil)
	case "submit":
		s.setSession(event.UserID, Session{Kind: SessionPhotoAfter, TaskID: task.ID})
		return s.send(ctx, event.ChatID, "📸 Прикрепите фотографию ПОСЛЕ выполнения работы:", nil)
	case "photos":
		return s.sendTaskPhotos(ctx, event, task, task.AfterPhotos, "📸 Фотоотчёт")
	case "before_photos":
		return s.sendTaskPhotos(ctx, event, task, task.BeforePhotos, "📷 Фото до начала работы")
	case "unable":
		s.setSession(event.UserID, Session{Kind: SessionUnableReason, TaskID: task.ID})
		return s.send(ctx, event.ChatID, "Напишите причину, по которой задание невозможно выполнить:", nil)
	case "accept":
		return s.reviewTask(ctx, event.ChatID, user, task, "accepted", "")
	case "rework":
		s.setSession(event.UserID, Session{Kind: SessionReworkComment, TaskID: task.ID})
		return s.send(ctx, event.ChatID, "🔁 Напишите, что именно нужно исправить:", nil)
	default:
		return s.taskCard(ctx, event, user, task)
	}
}
