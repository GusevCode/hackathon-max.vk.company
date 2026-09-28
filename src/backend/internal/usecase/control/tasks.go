package control

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/access"
)

func (s *Service) listTasks(ctx context.Context, event domain.Event, user domain.User) error {
	lines := []string{"📋 МОИ ЗАДАНИЯ\n\nНажмите на карточку для подробностей:"}
	buttons := make([]domain.Button, 0)
	for _, task := range s.repo.Tasks(user.OrganizationID) {
		if !access.CanViewTask(user, task) {
			continue
		}
		lines = append(lines, fmt.Sprintf("• %s %s — %s", taskStatusIcon(task.Status), taskStatusName(task.Status), task.Title))
		buttons = append(buttons, domain.Button{Text: taskStatusIcon(task.Status) + " " + shortID(task.ID), Payload: "task:view:" + task.ID, Row: len(buttons) / 2})
	}
	if len(buttons) == 0 {
		lines = append(lines, "\nПока заданий нет.")
	}
	buttons = append(buttons, domain.Button{Text: "↩️ Главное меню", Payload: "menu:home", Row: len(buttons)/2 + 1})
	return s.send(ctx, event.ChatID, strings.Join(lines, "\n"), buttons)
}

func (s *Service) taskCard(ctx context.Context, event domain.Event, user domain.User, task domain.Task) error {
	text := fmt.Sprintf("%s %s\n\n%s\n\nСтатус: %s %s\nСрок: %s\nФото: до — %d, после — %d", taskStatusIcon(task.Status), task.Title, task.Description, taskStatusIcon(task.Status), taskStatusName(task.Status), task.DueAt.Format("02.01.2006"), len(task.BeforePhotos), len(task.AfterPhotos))
	if task.Comment != "" {
		text += "\nКомментарий: " + task.Comment
	}
	buttons := make([]domain.Button, 0, 4)
	if user.HasRole(domain.RoleEmployee) && task.AssigneeID == user.ID {
		if task.Status == domain.TaskAssigned || task.Status == domain.TaskRework {
			buttons = append(buttons, domain.Button{Text: "▶️ Взять в работу", Payload: "task:take:" + task.ID, Row: 0})
		}
		if task.Status == domain.TaskInProgress || task.Status == domain.TaskRework {
			buttons = append(buttons,
				domain.Button{Text: "📷 Фото до", Payload: "task:before:" + task.ID, Row: 1},
				domain.Button{Text: "📸 Фото после", Payload: "task:submit:" + task.ID, Row: 1},
			)
		}
		if task.Status == domain.TaskAssigned || task.Status == domain.TaskInProgress || task.Status == domain.TaskRework {
			buttons = append(buttons, domain.Button{Text: "⚠️ Не могу выполнить", Payload: "task:unable:" + task.ID, Row: 2})
		}
	}
	if task.Status == domain.TaskSubmitted && access.CanReviewTask(user, task) {
		buttons = append(buttons,
			domain.Button{Text: "✅ Принять", Payload: "task:accept:" + task.ID, Row: 0},
			domain.Button{Text: "🔁 На переделку", Payload: "task:rework:" + task.ID, Row: 0},
		)
	}
	if len(task.AfterPhotos) > 0 {
		buttons = append(buttons, domain.Button{Text: "📸 Посмотреть фото", Payload: "task:photos:" + task.ID, Row: 2})
	}
	buttons = append(buttons, domain.Button{Text: "↩️ К заданиям", Payload: "menu:tasks", Row: 3})
	return s.send(ctx, event.ChatID, text, buttons)
}

func (s *Service) sendTaskPhotos(ctx context.Context, event domain.Event, task domain.Task) error {
	if len(task.AfterPhotos) == 0 {
		return s.send(ctx, event.ChatID, "📸 Для этого задания пока нет фотографий.", nil)
	}
	_, err := s.bot.Send(ctx, domain.OutgoingMessage{
		ChatID: event.ChatID,
		Text:   "📸 Фотоотчёт: " + task.Title,
		Photos: task.AfterPhotos,
		Buttons: []domain.Button{{
			Text:    "↩️ К заданию",
			Payload: "task:view:" + task.ID,
			Row:     0,
		}},
	})
	return err
}

func (s *Service) takeTask(ctx context.Context, chatID int64, user domain.User, task domain.Task) error {
	if !access.Can(user, access.ActionExecuteTask) || task.AssigneeID != user.ID || (task.Status != domain.TaskAssigned && task.Status != domain.TaskRework) {
		return s.sendHome(ctx, chatID, user, "Нельзя взять это задание в работу.")
	}
	task.Status = domain.TaskInProgress
	task.UpdatedAt = time.Now()
	s.repo.SaveTask(task)
	s.persistTask(ctx, task)
	return s.send(ctx, chatID, "▶️ Задание взято в работу. Теперь выполните его и отправьте фото после завершения.", menuForUser(user))
}

func (s *Service) reviewTask(ctx context.Context, chatID int64, user domain.User, task domain.Task, decision, comment string) error {
	if !access.CanReviewTask(user, task) || task.Status != domain.TaskSubmitted {
		return s.sendHome(ctx, chatID, user, "Это задание нельзя проверить сейчас.")
	}
	if decision == "rework" && strings.TrimSpace(comment) == "" {
		return s.send(ctx, chatID, "Комментарий для переделки не может быть пустым.", nil)
	}
	task.Status = domain.TaskAccepted
	if decision == "rework" {
		task.Status = domain.TaskRework
		task.Comment = strings.TrimSpace(comment)
	}
	task.UpdatedAt = time.Now()
	s.repo.SaveTask(task)
	s.persistTask(ctx, task)
	review := domain.Review{TaskID: task.ID, ReviewerID: user.ID, Decision: decision, Comment: task.Comment, CreatedAt: time.Now()}
	s.repo.SaveReview(review)
	if s.storage != nil {
		if err := s.storage.SaveReview(ctx, review); err != nil {
			s.logger.Warn("persist review", "error", err, "task_id", task.ID)
		}
	}
	if employee, ok := s.userByID(task.OrganizationID, task.AssigneeID); ok {
		message := "✅ Работа принята руководителем."
		if decision == "rework" {
			message = "🔁 Работа отправлена на переделку.\nЧто исправить: " + task.Comment
		}
		if err := s.notifyUser(ctx, employee.MaxUserID, domain.NotificationTaskReviewed, task.ID, message, menuForUser(employee)); err != nil {
			s.logger.Warn("notify employee", "error", err, "task_id", task.ID)
		}
	}
	s.clearSession(user.MaxUserID)
	return s.sendHome(ctx, chatID, user, "✅ Решение сохранено: "+decisionName(decision)+".")
}

func (s *Service) submitPhotos(ctx context.Context, event domain.Event, user domain.User, task domain.Task, photos []domain.Photo, comment string) error {
	if len(photos) == 0 {
		return s.send(ctx, event.ChatID, "Приложите хотя бы одну фотографию.", nil)
	}
	if strings.TrimSpace(comment) == "" {
		return s.send(ctx, event.ChatID, "Комментарий обязателен — напишите, что именно сделано.", nil)
	}
	task.AfterPhotos = append(task.AfterPhotos, photos...)
	if s.photos != nil {
		if err := s.persistPhotos(ctx, task.ID, "after", photos); err != nil {
			s.logger.Warn("upload after photos", "error", err, "task_id", task.ID, "photos", len(photos))
			return s.send(ctx, event.ChatID, "Не удалось сохранить фото. Попробуйте ещё раз.", nil)
		}
	}
	task.Comment = strings.TrimSpace(comment)
	task.Status = domain.TaskSubmitted
	task.UpdatedAt = time.Now()
	s.repo.SaveTask(task)
	s.persistTask(ctx, task)
	s.clearSession(event.UserID)
	if manager, ok := s.userByID(task.OrganizationID, task.ManagerID); ok {
		if err := s.notifyUser(ctx, manager.MaxUserID, domain.NotificationTaskSubmitted, task.ID, fmt.Sprintf("📸 Новый фотоотчёт\n\n%s\nКомментарий: %s", task.Title, task.Comment), menuForUser(manager), task.AfterPhotos); err != nil {
			s.logger.Warn("notify manager", "error", err, "task_id", task.ID)
		}
	}
	return s.sendHome(ctx, event.ChatID, user, "📸 Фотоотчёт отправлен руководителю на проверку.")
}

func (s *Service) attachBefore(ctx context.Context, event domain.Event, user domain.User, task domain.Task, photos []domain.Photo) error {
	if len(photos) == 0 {
		return s.send(ctx, event.ChatID, "Приложите фотографию до начала работы.", nil)
	}
	task.BeforePhotos = append(task.BeforePhotos, photos...)
	if s.photos != nil {
		if err := s.persistPhotos(ctx, task.ID, "before", photos); err != nil {
			s.logger.Warn("upload before photos", "error", err, "task_id", task.ID, "photos", len(photos))
			return s.send(ctx, event.ChatID, "Не удалось сохранить фото. Попробуйте ещё раз.", nil)
		}
	}
	task.Status = domain.TaskInProgress
	task.UpdatedAt = time.Now()
	s.repo.SaveTask(task)
	s.persistTask(ctx, task)
	s.clearSession(event.UserID)
	return s.sendHome(ctx, event.ChatID, user, "📷 Фото до начала сохранено.")
}

func (s *Service) markUnable(ctx context.Context, event domain.Event, user domain.User, task domain.Task, reason string) error {
	if !access.Can(user, access.ActionExecuteTask) || task.AssigneeID != user.ID || strings.TrimSpace(reason) == "" {
		return s.sendHome(ctx, event.ChatID, user, "Нужно указать причину.")
	}
	task.Status = domain.TaskUnable
	task.Comment = strings.TrimSpace(reason)
	task.UpdatedAt = time.Now()
	s.repo.SaveTask(task)
	s.persistTask(ctx, task)
	s.clearSession(event.UserID)
	return s.sendHome(ctx, event.ChatID, user, "⚠️ Причина сохранена и передана руководителю.")
}

func taskStatusIcon(status domain.TaskStatus) string {
	switch status {
	case domain.TaskAssigned:
		return "🟡"
	case domain.TaskInProgress:
		return "🔵"
	case domain.TaskSubmitted:
		return "🟣"
	case domain.TaskAccepted:
		return "✅"
	case domain.TaskRework:
		return "🔁"
	case domain.TaskUnable:
		return "⚠️"
	default:
		return "⚪"
	}
}

func taskStatusName(status domain.TaskStatus) string {
	switch status {
	case domain.TaskAssigned:
		return "Назначено"
	case domain.TaskInProgress:
		return "В работе"
	case domain.TaskSubmitted:
		return "На проверке"
	case domain.TaskAccepted:
		return "Принято"
	case domain.TaskRework:
		return "Переделка"
	case domain.TaskUnable:
		return "Невозможно"
	default:
		return "Неизвестно"
	}
}

func decisionName(decision string) string {
	if decision == "rework" {
		return "на переделку"
	}
	return "принято"
}

func (s *Service) userByID(organizationID, id string) (domain.User, bool) {
	for _, user := range s.repo.Users(organizationID) {
		if user.ID == id {
			return user, true
		}
	}
	return domain.User{}, false
}
