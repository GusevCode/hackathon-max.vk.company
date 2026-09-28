package control

import (
	"context"
	"fmt"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

func (s *Service) handleClearCallback(ctx context.Context, event domain.Event, user domain.User, action string) error {
	if !user.HasRole(domain.RoleAdmin) {
		return s.sendHome(ctx, event.ChatID, user, "Только администратор может очищать систему.")
	}
	switch action {
	case "yes":
		return s.clearOrganizationData(ctx, event, user)
	case "no":
		return s.adminMenu(ctx, event.ChatID, user)
	default:
		return s.send(ctx, event.ChatID, "Подтвердите действие кнопкой ниже.", []domain.Button{
			{Text: "🧹 Очистить задания и фото", Payload: "admin:clear:yes", Row: 0},
			{Text: "Отмена", Payload: "admin:clear:no", Row: 1},
		})
	}
}

func (s *Service) clearOrganizationData(ctx context.Context, event domain.Event, user domain.User) error {
	if !user.HasRole(domain.RoleAdmin) {
		return s.sendHome(ctx, event.ChatID, user, "Недостаточно прав.")
	}
	if s.photos != nil {
		if err := s.photos.DeleteAllTaskPhotos(ctx); err != nil {
			s.logger.Error("clear task photos", "error", err)
			return s.sendHome(ctx, event.ChatID, user, "Не удалось удалить фотографии. Задания не были очищены.")
		}
	}
	if s.storage != nil {
		if err := s.storage.ClearTasks(ctx, user.OrganizationID); err != nil {
			s.logger.Error("clear persisted tasks", "error", err)
			return s.sendHome(ctx, event.ChatID, user, "Не удалось очистить задания в базе данных.")
		}
	}
	deleted := s.repo.ClearTasks(user.OrganizationID)
	return s.sendHome(ctx, event.ChatID, user, fmt.Sprintf("✅ Система очищена. Удалено заданий: %d. Сотрудники и пользователи сохранены.", len(deleted)))
}
