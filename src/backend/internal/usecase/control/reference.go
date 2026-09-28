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
	return s.send(ctx, event.ChatID, strings.Join(lines, "\n"), []domain.Button{
		{Text: "➕ Добавить объект", Payload: "reference:create_object", Row: 0},
		{Text: "↩️ Главное меню", Payload: "menu:home", Row: 1},
	})
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
	return s.send(ctx, event.ChatID, strings.Join(lines, "\n"), []domain.Button{
		{Text: "➕ Добавить вид работы", Payload: "reference:create_worktype", Row: 0},
		{Text: "↩️ Главное меню", Payload: "menu:home", Row: 1},
	})
}

func (s *Service) handleReferenceCallback(ctx context.Context, event domain.Event, user domain.User, target string) error {
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
	default:
		return s.sendHome(ctx, event.ChatID, user, "Раздел не найден.")
	}
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
