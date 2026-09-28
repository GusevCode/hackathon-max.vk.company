package control

import (
	"context"
	"fmt"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/access"
)

func (s *Service) createInvite(ctx context.Context, event domain.Event, user domain.User, role domain.Role) error {
	if !role.Valid() || role == domain.RoleAdmin || !access.CanInviteRole(user, role) {
		return s.sendHome(ctx, event.ChatID, user, "Нельзя создать приглашение для этой роли.")
	}
	invite := domain.Invite{Code: newCode(), OrganizationID: user.OrganizationID, Roles: []domain.Role{role}, ExpiresAt: time.Now().Add(24 * time.Hour)}
	s.repo.SaveInvite(invite)
	if s.storage != nil {
		if err := s.storage.SaveInvite(ctx, invite); err != nil {
			s.logger.Warn("persist invite", "error", err)
		}
	}
	return s.send(ctx, event.ChatID, fmt.Sprintf("🎟 ПРИГЛАСИТЕЛЬНЫЙ КОД\n\nРоль: %s\nКод: %s\nСрок действия: 24 часа\n\nПередайте код новому пользователю — он введёт его в меню бота.", role.Label(), invite.Code), []domain.Button{
		{Text: "🎟 Создать ещё код", Payload: "admin:invite", Row: 0},
		{Text: "↩️ Управление", Payload: "menu:admin", Row: 1},
	})
}
