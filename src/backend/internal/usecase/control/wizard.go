package control

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/access"
)

func (s *Service) handleSessionInput(ctx context.Context, event domain.Event, user domain.User) error {
	session, ok := s.session(event.UserID)
	if !ok {
		return s.sendHome(ctx, event.ChatID, user, "Сценарий завершён.")
	}
	text := strings.TrimSpace(event.Text)
	switch session.Kind {
	case SessionTaskTitle:
		if text == "" {
			return s.send(ctx, event.ChatID, "Введите название текстом.", nil)
		}
		session.Draft.Title = text
		session.Kind = SessionTaskDescription
		s.setSession(event.UserID, session)
		return s.send(ctx, event.ChatID, "📝 Добавьте краткое описание работы:", nil)
	case SessionTaskDescription:
		if text == "" {
			return s.send(ctx, event.ChatID, "Описание не должно быть пустым.", nil)
		}
		session.Draft.Description = text
		session.Kind = SessionTaskDueDate
		s.setSession(event.UserID, session)
		return s.chooseEmployee(ctx, event, user, session)
	case SessionTaskDueDate:
		due, err := time.Parse("02.01.2006", text)
		if err != nil || due.Before(time.Now().AddDate(0, 0, -1)) {
			return s.send(ctx, event.ChatID, "Введите будущую дату в формате ДД.ММ.ГГГГ, например 25.10.2026.", nil)
		}
		session.Draft.DueAt = due.Format("02.01.2006")
		s.setSession(event.UserID, session)
		return s.chooseObject(ctx, event, user, session)
	case SessionObjectName:
		if !access.Can(user, access.ActionManageObjects) {
			return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
		}
		if text == "" {
			return s.send(ctx, event.ChatID, "Введите название объекта.", nil)
		}
		session.ObjectName = text
		session.Kind = SessionObjectAddress
		s.setSession(event.UserID, session)
		return s.send(ctx, event.ChatID, "Укажите адрес объекта:", nil)
	case SessionObjectAddress:
		if text == "" {
			return s.send(ctx, event.ChatID, "Адрес не должен быть пустым.", nil)
		}
		return s.saveObject(ctx, event, user, session.ObjectName, text)
	case SessionWorkTypeName:
		if !access.Can(user, access.ActionManageWorkType) {
			return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
		}
		if text == "" {
			return s.send(ctx, event.ChatID, "Введите название вида работы.", nil)
		}
		return s.saveWorkType(ctx, event, user, text)
	case SessionPhotoBefore:
		task, ok := s.repo.Task(session.TaskID)
		if !ok {
			return s.sendHome(ctx, event.ChatID, user, "Задание не найдено.")
		}
		return s.attachBefore(ctx, event, user, task, event.Photos)
	case SessionPhotoAfter:
		task, ok := s.repo.Task(session.TaskID)
		if !ok {
			return s.sendHome(ctx, event.ChatID, user, "Задание не найдено.")
		}
		if len(event.Photos) == 0 {
			return s.send(ctx, event.ChatID, "Нужно прикрепить фотографию после выполнения.", nil)
		}
		if text != "" {
			return s.submitPhotos(ctx, event, user, task, event.Photos, text)
		}
		session.Kind = SessionPhotoAfterRemark
		session.Photos = event.Photos
		s.setSession(event.UserID, session)
		return s.send(ctx, event.ChatID, "Фото получено. Теперь напишите комментарий к выполненной работе:", nil)
	case SessionPhotoAfterRemark:
		task, ok := s.repo.Task(session.TaskID)
		if !ok {
			return s.sendHome(ctx, event.ChatID, user, "Задание не найдено.")
		}
		return s.submitPhotos(ctx, event, user, task, session.Photos, text)
	case SessionUnableReason:
		task, ok := s.repo.Task(session.TaskID)
		if !ok {
			return s.sendHome(ctx, event.ChatID, user, "Задание не найдено.")
		}
		return s.markUnable(ctx, event, user, task, text)
	case SessionReworkComment:
		task, ok := s.repo.Task(session.TaskID)
		if !ok {
			return s.sendHome(ctx, event.ChatID, user, "Задание не найдено.")
		}
		return s.reviewTask(ctx, event.ChatID, user, task, "rework", text)
	default:
		return s.sendHome(ctx, event.ChatID, user, "Сценарий не найден.")
	}
}

func (s *Service) chooseEmployee(ctx context.Context, event domain.Event, user domain.User, session Session) error {
	buttons := make([]domain.Button, 0)
	for _, employee := range s.repo.Users(user.OrganizationID) {
		if !employee.HasRole(domain.RoleEmployee) || employee.Status == domain.UserBlocked {
			continue
		}
		if user.HasRole(domain.RoleManager) && !user.HasRole(domain.RoleOperator) && !user.HasRole(domain.RoleAdmin) && employee.ManagerID != user.ID {
			continue
		}
		buttons = append(buttons, domain.Button{Text: "👷 " + shortName(employee.DisplayName), Payload: "wizard:employee:" + employee.ID, Row: len(buttons) / 2})
	}
	if len(buttons) == 0 {
		return s.sendHome(ctx, event.ChatID, user, "Нет доступных сотрудников для назначения.")
	}
	return s.send(ctx, event.ChatID, "👷 Выберите сотрудника:", buttons)
}

func (s *Service) chooseObject(ctx context.Context, event domain.Event, user domain.User, session Session) error {
	buttons := []domain.Button{{Text: "Без указания объекта", Payload: "wizard:object:none", Row: 0}}
	for _, object := range s.repo.Objects(user.OrganizationID) {
		buttons = append(buttons, domain.Button{Text: "🏢 " + shortName(object.Name), Payload: "wizard:object:" + object.ID, Row: len(buttons) / 2})
	}
	return s.send(ctx, event.ChatID, "🏢 Выберите объект или пропустите этот шаг:", buttons)
}

func (s *Service) chooseWorkType(ctx context.Context, event domain.Event, user domain.User, session Session) error {
	buttons := []domain.Button{{Text: "Без указания вида работы", Payload: "wizard:worktype:none", Row: 0}}
	for _, workType := range s.repo.WorkTypes(user.OrganizationID) {
		buttons = append(buttons, domain.Button{Text: "🧹 " + shortName(workType.Name), Payload: "wizard:worktype:" + workType.ID, Row: len(buttons) / 2})
	}
	return s.send(ctx, event.ChatID, "🧹 Выберите вид работы или пропустите этот шаг:", buttons)
}

func (s *Service) confirmTask(ctx context.Context, event domain.Event, user domain.User, session Session) error {
	employee, _ := s.userByID(user.OrganizationID, session.Draft.AssigneeID)
	return s.send(ctx, event.ChatID, fmt.Sprintf("ПРОВЕРЬТЕ ЗАДАНИЕ\n\n%s\n%s\nСотрудник: %s\nСрок: %s\n\nСоздать задание?", session.Draft.Title, session.Draft.Description, employee.DisplayName, session.Draft.DueAt), []domain.Button{
		{Text: "✅ Создать", Payload: "wizard:confirm:yes", Row: 0},
		{Text: "✖️ Отменить", Payload: "wizard:confirm:no", Row: 0},
	})
}

func (s *Service) createTaskFromDraft(ctx context.Context, event domain.Event, user domain.User, session Session) error {
	if !access.Can(user, access.ActionCreateTask) {
		return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
	}
	employee, ok := s.userByID(user.OrganizationID, session.Draft.AssigneeID)
	if !ok || !employee.HasRole(domain.RoleEmployee) {
		return s.sendHome(ctx, event.ChatID, user, "Сотрудник больше недоступен.")
	}
	if user.HasRole(domain.RoleManager) && !user.HasRole(domain.RoleOperator) && !user.HasRole(domain.RoleAdmin) && employee.ManagerID != user.ID {
		return s.sendHome(ctx, event.ChatID, user, "Сотрудник не закреплён за вами.")
	}
	due, _ := time.Parse("02.01.2006", session.Draft.DueAt)
	now := time.Now()
	task := domain.Task{ID: newCode(), OrganizationID: user.OrganizationID, Title: session.Draft.Title, Description: session.Draft.Description, AssigneeID: employee.ID, ManagerID: user.ID, ObjectID: normalizedOptional(session.Draft.ObjectID), WorkTypeID: normalizedOptional(session.Draft.WorkTypeID), DueAt: due, Priority: domain.PriorityNormal, Status: domain.TaskAssigned, CreatedAt: now, UpdatedAt: now}
	s.repo.SaveTask(task)
	s.persistTask(ctx, task)
	s.clearSession(event.UserID)
	if err := s.send(ctx, employee.MaxUserID, fmt.Sprintf("📌 Вам назначено новое задание\n\n%s\nСрок: %s", task.Title, task.DueAt.Format("02.01.2006")), menuForUser(employee)); err != nil {
		s.logger.Warn("notify employee", "error", err, "task_id", task.ID)
	}
	return s.sendHome(ctx, event.ChatID, user, "✅ Задание создано и отправлено сотруднику.")
}

func normalizedOptional(value string) string {
	if value == "none" {
		return ""
	}
	return value
}
