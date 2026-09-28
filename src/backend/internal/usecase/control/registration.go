package control

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

func (s *Service) handleRegistrationInput(ctx context.Context, event domain.Event) error {
	session, ok := s.session(event.UserID)
	if !ok || session.Kind != SessionInviteCode {
		return s.showRegistration(ctx, event.ChatID)
	}
	code := strings.ToUpper(strings.TrimSpace(event.Text))
	invite, exists := s.repo.Invite(code)
	if !exists || !invite.IsAvailable(time.Now()) {
		return s.send(ctx, event.ChatID, "❌ Код недействителен или уже использован. Проверьте код и попробуйте ещё раз.", []domain.Button{{Text: "🎟 Ввести другой код", Payload: "auth:invite", Row: 0}})
	}
	user := domain.User{
		ID:             fmt.Sprintf("user-%d", event.UserID),
		OrganizationID: invite.OrganizationID,
		MaxUserID:      event.UserID,
		DisplayName:    displayName(event.DisplayName, event.UserID),
		Roles:          invite.Roles,
		Status:         domain.UserActive,
		ManagerID:      invite.ManagerID,
		CreatedAt:      time.Now(),
	}
	s.repo.SaveUser(user)
	s.persistUser(ctx, user)
	invite.UsedBy = event.UserID
	s.repo.SaveInvite(invite)
	if s.storage != nil {
		if err := s.storage.SaveInvite(ctx, invite); err != nil {
			s.logger.Warn("persist invite", "error", err)
		}
	}
	s.clearSession(event.UserID)
	return s.sendHome(ctx, event.ChatID, user, "✅ Регистрация завершена!")
}

func displayName(name string, userID int64) string {
	if strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return fmt.Sprintf("Пользователь %d", userID)
}
