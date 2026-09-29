package control

import (
	"context"
	"fmt"
	"strconv"
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
		buttons = append(buttons, domain.Button{
			Text:    fmt.Sprintf("%s %s (%s)", taskStatusIcon(task.Status), task.Title, shortID(task.ID)),
			Payload: "task:view:" + task.ID,
			Row:     len(buttons),
		})
	}
	if len(buttons) == 0 {
		lines = append(lines, "\nПока заданий нет.")
	}
	buttons = append(buttons, domain.Button{Text: "↩️ Главное меню", Payload: "menu:home", Row: len(buttons)})
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
		text += s.analysisText(task)
		buttons = append(buttons,
			domain.Button{Text: "✅ Принять", Payload: "task:accept:" + task.ID, Row: 0},
			domain.Button{Text: "🔁 На переделку", Payload: "task:rework:" + task.ID, Row: 0},
		)
		if s.inspections != nil {
			buttons = append(buttons, domain.Button{Text: "🤖 Повторить ИИ-анализ", Payload: "task:reanalyze:" + task.ID, Row: 1})
		}
	}
	if access.CanEditTask(user, task) {
		buttons = append(buttons, domain.Button{Text: "✏️ Редактировать", Payload: "task:edit:" + task.ID, Row: 2})
	}
	if access.CanCloseTask(user, task) {
		buttons = append(buttons, domain.Button{Text: "🗄 Закрыть задание", Payload: "task:close:" + task.ID, Row: 3})
	}
	if len(task.BeforePhotos) > 0 {
		buttons = append(buttons, domain.Button{Text: "📷 Посмотреть фото до", Payload: "task:before_photos:" + task.ID, Row: 2})
	}
	if len(task.AfterPhotos) > 0 {
		buttons = append(buttons, domain.Button{Text: "📸 Посмотреть фото", Payload: "task:photos:" + task.ID, Row: 2})
	}
	buttons = append(buttons, domain.Button{Text: "↩️ К заданиям", Payload: "menu:tasks", Row: 4})
	return s.send(ctx, event.ChatID, text, buttons)
}

func (s *Service) closeTask(ctx context.Context, chatID int64, user domain.User, task domain.Task) error {
	if !access.CanCloseTask(user, task) {
		return s.sendHome(ctx, chatID, user, "Это задание нельзя закрыть.")
	}
	task.Status = domain.TaskClosed
	task.UpdatedAt = time.Now()
	s.repo.SaveTask(task)
	s.persistTask(ctx, task)

	if employee, ok := s.userByID(task.OrganizationID, task.AssigneeID); ok {
		message := fmt.Sprintf("🗄 Задание закрыто\n\n%s (%s)", task.Title, shortID(task.ID))
		if err := s.notifyUser(ctx, employee.MaxUserID, domain.NotificationTaskClosed, task.ID, message, menuForUser(employee)); err != nil {
			s.logger.Warn("notify employee about closed task", "error", err, "task_id", task.ID)
		}
	}

	s.clearSession(user.MaxUserID)
	return s.sendHome(ctx, chatID, user, "🗄 Задание закрыто и убрано из активных списков.")
}

func (s *Service) sendTaskPhotos(ctx context.Context, event domain.Event, user domain.User, task domain.Task, photos []domain.Photo, kind, label string) error {
	if len(photos) == 0 {
		return s.send(ctx, event.ChatID, "📷 Для этого задания пока нет фотографий.", nil)
	}
	buttons := make([]domain.Button, 0, len(photos)+1)
	if access.CanDeleteTaskPhoto(user, task) {
		for index := range photos {
			buttons = append(buttons, domain.Button{
				Text:    fmt.Sprintf("🗑 Удалить фото %d", index+1),
				Payload: fmt.Sprintf("task:delete_photo:%s:%s:%d", task.ID, kind, index),
				Row:     index,
			})
		}
	}
	buttons = append(buttons, domain.Button{Text: "↩️ К заданию", Payload: "task:view:" + task.ID, Row: len(buttons)})
	_, err := s.bot.Send(ctx, domain.OutgoingMessage{
		ChatID:  event.ChatID,
		Text:    label + ": " + task.Title,
		Photos:  photos,
		Buttons: buttons,
	})
	return err
}

func (s *Service) handleTaskPhotoCallback(ctx context.Context, event domain.Event, user domain.User, action, taskID, kind, rawIndex string) error {
	if kind != "before" && kind != "after" {
		return s.sendHome(ctx, event.ChatID, user, "Раздел фотографий не найден.")
	}
	task, ok := s.repo.Task(taskID)
	if !ok || !access.CanViewTask(user, task) || !access.CanDeleteTaskPhoto(user, task) {
		return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав или задание недоступно.")
	}
	index, err := strconv.Atoi(rawIndex)
	if err != nil {
		return s.sendHome(ctx, event.ChatID, user, "Фотография не найдена.")
	}
	photos := taskPhotos(task, kind)
	if index < 0 || index >= len(photos) {
		return s.sendHome(ctx, event.ChatID, user, "Фотография не найдена.")
	}
	if action == "delete_photo" {
		return s.send(ctx, event.ChatID, fmt.Sprintf("🗑 Удалить фото %d из раздела «%s»?", index+1, photoKindLabel(kind)), []domain.Button{
			{Text: "✅ Удалить", Payload: fmt.Sprintf("task:delete_photo_confirm:%s:%s:%d", task.ID, kind, index), Row: 0},
			{Text: "↩️ Отмена", Payload: fmt.Sprintf("task:%s:%s", photoListAction(kind), task.ID), Row: 1},
		})
	}

	photo := photos[index]
	if photo.ObjectKey != "" {
		if s.photos != nil {
			if err := s.photos.DeleteObject(ctx, photo.ObjectKey); err != nil {
				s.logger.Warn("delete task photo", "error", err, "task_id", task.ID, "object_key", photo.ObjectKey)
				return s.send(ctx, event.ChatID, "Не удалось удалить фотографию из хранилища. Попробуйте ещё раз.", []domain.Button{{
					Text: "↩️ К фотографиям", Payload: fmt.Sprintf("task:%s:%s", photoListAction(kind), task.ID), Row: 0,
				}})
			}
		}
		if s.storage != nil {
			if err := s.storage.DeleteEvidence(ctx, task.ID, photo.ObjectKey); err != nil {
				s.logger.Warn("delete task photo evidence", "error", err, "task_id", task.ID, "object_key", photo.ObjectKey)
			}
		}
	}

	if kind == "before" {
		task.BeforePhotos = append(task.BeforePhotos[:index], task.BeforePhotos[index+1:]...)
	} else {
		task.AfterPhotos = append(task.AfterPhotos[:index], task.AfterPhotos[index+1:]...)
		if len(task.AfterPhotos) == 0 && (task.Status == domain.TaskSubmitted || task.Status == domain.TaskAccepted) {
			task.Status = domain.TaskInProgress
		}
	}
	task.UpdatedAt = time.Now()
	s.repo.SaveTask(task)
	s.persistTask(ctx, task)
	return s.sendTaskPhotos(ctx, event, user, task, taskPhotos(task, kind), kind, photoKindLabel(kind))
}

func taskPhotos(task domain.Task, kind string) []domain.Photo {
	if kind == "before" {
		return task.BeforePhotos
	}
	return task.AfterPhotos
}

func photoKindLabel(kind string) string {
	if kind == "before" {
		return "📷 Фото до начала работы"
	}
	return "📸 Фотоотчёт"
}

func photoListAction(kind string) string {
	if kind == "before" {
		return "before_photos"
	}
	return "photos"
}

func (s *Service) taskEditMenu(ctx context.Context, event domain.Event, user domain.User, task domain.Task) error {
	if !access.CanEditTask(user, task) {
		return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав для редактирования задания.")
	}
	return s.send(ctx, event.ChatID, "✏️ Что изменить в задании?", []domain.Button{
		{Text: "📝 Название", Payload: "task:edit_title:" + task.ID, Row: 0},
		{Text: "📄 Описание", Payload: "task:edit_description:" + task.ID, Row: 1},
		{Text: "📅 Срок", Payload: "task:edit_due:" + task.ID, Row: 2},
		{Text: "↩️ К заданию", Payload: "task:view:" + task.ID, Row: 3},
	})
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
	submissionID := newCode()
	var afterEvidence []domain.Evidence
	if s.photos != nil {
		var err error
		afterEvidence, err = s.persistPhotos(ctx, task.ID, "after", submissionID, photos)
		if err != nil {
			s.logger.Warn("upload after photos", "error", err, "task_id", task.ID, "photos", len(photos))
			return s.send(ctx, event.ChatID, "Не удалось сохранить фото. Попробуйте ещё раз.", nil)
		}
	}
	task.AfterPhotos = append(task.AfterPhotos, photos...)
	task.SubmissionID = submissionID
	task.Comment = strings.TrimSpace(comment)
	task.Status = domain.TaskSubmitted
	task.UpdatedAt = time.Now()
	s.repo.SaveTask(task)
	s.persistTask(ctx, task)
	s.requestInspection(ctx, task, afterEvidence)
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
	if s.photos != nil {
		if _, err := s.persistPhotos(ctx, task.ID, "before", "", photos); err != nil {
			s.logger.Warn("upload before photos", "error", err, "task_id", task.ID, "photos", len(photos))
			return s.send(ctx, event.ChatID, "Не удалось сохранить фото. Попробуйте ещё раз.", nil)
		}
	}
	task.BeforePhotos = append(task.BeforePhotos, photos...)
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
	case domain.TaskClosed:
		return "🗄"
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
	case domain.TaskClosed:
		return "Закрыто"
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
