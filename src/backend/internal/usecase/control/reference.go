package control

import (
	"context"
	"fmt"
	"strings"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/access"
)

func (s *Service) listObjects(ctx context.Context, event domain.Event, user domain.User) error {
	if !access.Can(user, access.ActionManageObjects) {
		return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
	}
	lines := []string{"🏢 ОБЪЕКТЫ\n"}
	for _, object := range s.repo.Objects(user.OrganizationID) {
		lines = append(lines, fmt.Sprintf("• %s — %s", object.Name, object.Address))
	}
	if len(lines) == 1 {
		lines = append(lines, "Пока объектов нет.")
	}
	buttons := make([]domain.Button, 0, len(s.repo.Objects(user.OrganizationID))+2)
	for _, object := range s.repo.Objects(user.OrganizationID) {
		buttons = append(buttons, domain.Button{Text: "✏️ " + shortName(object.Name), Payload: "reference:edit_object:" + object.ID, Row: len(buttons)})
	}
	buttons = append(buttons,
		domain.Button{Text: "➕ Добавить объект", Payload: "reference:create_object", Row: len(buttons)},
		domain.Button{Text: "↩️ Главное меню", Payload: "menu:home", Row: len(buttons) + 1},
	)
	return s.send(ctx, event.ChatID, strings.Join(lines, "\n"), buttons)
}

func (s *Service) listWorkTypes(ctx context.Context, event domain.Event, user domain.User) error {
	if !access.Can(user, access.ActionManageWorkType) {
		return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
	}
	lines := []string{"🧹 ВИДЫ РАБОТ\n"}
	for _, workType := range s.repo.WorkTypes(user.OrganizationID) {
		lines = append(lines, "• "+workType.Name)
	}
	if len(lines) == 1 {
		lines = append(lines, "Пока видов работ нет.")
	}
	buttons := make([]domain.Button, 0, len(s.repo.WorkTypes(user.OrganizationID))+2)
	for _, workType := range s.repo.WorkTypes(user.OrganizationID) {
		buttons = append(buttons, domain.Button{Text: "✏️ " + shortName(workType.Name), Payload: "reference:edit_worktype:" + workType.ID, Row: len(buttons)})
	}
	buttons = append(buttons,
		domain.Button{Text: "➕ Добавить вид работы", Payload: "reference:create_worktype", Row: len(buttons)},
		domain.Button{Text: "↩️ Главное меню", Payload: "menu:home", Row: len(buttons) + 1},
	)
	return s.send(ctx, event.ChatID, strings.Join(lines, "\n"), buttons)
}

func (s *Service) handleReferenceCallback(ctx context.Context, event domain.Event, user domain.User, target string, ids ...string) error {
	refID := ""
	if len(ids) > 0 {
		refID = ids[0]
	}
	switch target {
	case "create_object":
		if !access.Can(user, access.ActionManageObjects) {
			return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
		}
		s.setSession(event.UserID, Session{Kind: SessionObjectName})
		return s.send(ctx, event.ChatID, "🏢 Введите название объекта:", nil)
	case "create_worktype":
		if !access.Can(user, access.ActionManageWorkType) {
			return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
		}
		s.setSession(event.UserID, Session{Kind: SessionWorkTypeName})
		return s.send(ctx, event.ChatID, "🧹 Введите название нового вида работы:", nil)
	case "edit_object":
		if !access.Can(user, access.ActionManageObjects) {
			return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
		}
		object, ok := s.referenceObject(user.OrganizationID, refID)
		if !ok {
			return s.sendHome(ctx, event.ChatID, user, "Объект не найден.")
		}
		s.setSession(event.UserID, Session{Kind: SessionObjectEditName, ReferenceID: object.ID, ObjectName: object.Name})
		return s.send(ctx, event.ChatID, "🏢 Введите новое название объекта:", nil)
	case "edit_worktype":
		if !access.Can(user, access.ActionManageWorkType) {
			return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
		}
		workType, ok := s.referenceWorkType(user.OrganizationID, refID)
		if !ok {
			return s.sendHome(ctx, event.ChatID, user, "Вид работы не найден.")
		}
		s.setSession(event.UserID, Session{Kind: SessionWorkTypeEditName, ReferenceID: workType.ID})
		return s.send(ctx, event.ChatID, "🧹 Введите новое название вида работы:", nil)
	default:
		return s.sendHome(ctx, event.ChatID, user, "Раздел не найден.")
	}
}

func (s *Service) referenceObject(organizationID, id string) (domain.Object, bool) {
	for _, object := range s.repo.Objects(organizationID) {
		if object.ID == id {
			return object, true
		}
	}
	return domain.Object{}, false
}

func (s *Service) referenceWorkType(organizationID, id string) (domain.WorkType, bool) {
	for _, workType := range s.repo.WorkTypes(organizationID) {
		if workType.ID == id {
			return workType, true
		}
	}
	return domain.WorkType{}, false
}

func (s *Service) saveObject(ctx context.Context, event domain.Event, user domain.User, name, address string) error {
	if !access.Can(user, access.ActionManageObjects) {
		return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
	}
	object := domain.Object{ID: newCode(), OrganizationID: user.OrganizationID, Name: strings.TrimSpace(name), Address: strings.TrimSpace(address), Kind: "housing"}
	s.repo.SaveObject(object)
	if s.storage != nil {
		if err := s.storage.SaveObject(ctx, object); err != nil {
			s.logger.Warn("persist object", "error", err, "object_id", object.ID)
		}
	}
	s.clearSession(event.UserID)
	return s.sendHome(ctx, event.ChatID, user, "✅ Объект добавлен.")
}

func (s *Service) saveWorkType(ctx context.Context, event domain.Event, user domain.User, name string) error {
	if !access.Can(user, access.ActionManageWorkType) {
		return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
	}
	workType := domain.WorkType{ID: newCode(), OrganizationID: user.OrganizationID, Name: strings.TrimSpace(name)}
	s.repo.SaveWorkType(workType)
	if s.storage != nil {
		if err := s.storage.SaveWorkType(ctx, workType); err != nil {
			s.logger.Warn("persist work type", "error", err, "work_type_id", workType.ID)
		}
	}
	s.clearSession(event.UserID)
	return s.sendHome(ctx, event.ChatID, user, "✅ Вид работы добавлен.")
}

func (s *Service) updateObject(ctx context.Context, event domain.Event, user domain.User, session Session, address string) error {
	if !access.Can(user, access.ActionManageObjects) {
		return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
	}
	object, ok := s.referenceObject(user.OrganizationID, session.ReferenceID)
	if !ok {
		return s.sendHome(ctx, event.ChatID, user, "Объект не найден.")
	}
	object.Name = strings.TrimSpace(session.ObjectName)
	object.Address = strings.TrimSpace(address)
	s.repo.SaveObject(object)
	if s.storage != nil {
		if err := s.storage.SaveObject(ctx, object); err != nil {
			s.logger.Warn("persist object update", "error", err, "object_id", object.ID)
		}
	}
	s.clearSession(event.UserID)
	return s.sendHome(ctx, event.ChatID, user, "✅ Объект изменён.")
}

func (s *Service) updateWorkType(ctx context.Context, event domain.Event, user domain.User, session Session, name string) error {
	if !access.Can(user, access.ActionManageWorkType) {
		return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
	}
	workType, ok := s.referenceWorkType(user.OrganizationID, session.ReferenceID)
	if !ok {
		return s.sendHome(ctx, event.ChatID, user, "Вид работы не найден.")
	}
	workType.Name = strings.TrimSpace(name)
	s.repo.SaveWorkType(workType)
	if s.storage != nil {
		if err := s.storage.SaveWorkType(ctx, workType); err != nil {
			s.logger.Warn("persist work type update", "error", err, "work_type_id", workType.ID)
		}
	}
	s.clearSession(event.UserID)
	return s.sendHome(ctx, event.ChatID, user, "✅ Вид работы изменён.")
}
