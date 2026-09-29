package control

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type Service struct {
	bot           BotGateway
	repo          Repository
	photos        PhotoStore
	storage       Storage
	notifications NotificationPublisher
	inspections   InspectionPublisher
	promptVersion string
	logger        *slog.Logger
	inviteCodeTTL time.Duration

	sessionsMu sync.Mutex
	sessions   map[int64]Session
	targets    map[int64]renderTarget
}

func (s *Service) SetNotificationPublisher(publisher NotificationPublisher) {
	s.notifications = publisher
}

func (s *Service) SetInspectionPublisher(publisher InspectionPublisher, promptVersion string) {
	s.inspections = publisher
	s.promptVersion = promptVersion
}

type renderTargetContextKey struct{}
type replaceRenderTargetContextKey struct{}

type renderTarget struct {
	chatID    int64
	messageID string
	userID    int64
}

func NewService(bot BotGateway, repo Repository, photos PhotoStore, logger *slog.Logger, storages ...Storage) *Service {
	var storage Storage
	if len(storages) > 0 {
		storage = storages[0]
	}
	return &Service{
		bot: bot, repo: repo, photos: photos, storage: storage, logger: logger,
		sessions: make(map[int64]Session), targets: make(map[int64]renderTarget),
		inviteCodeTTL: 24 * time.Hour,
	}
}

func (s *Service) SetInviteCodeTTL(ttl time.Duration) {
	if ttl > 0 {
		s.inviteCodeTTL = ttl
	}
}

func (s *Service) Run(ctx context.Context) error {
	info, err := s.bot.GetInfo(ctx)
	if err != nil {
		return fmt.Errorf("get MAX bot info: %w", err)
	}
	s.logger.Info("MAX bot connected", "name", info.Name, "username", info.Username, "user_id", info.ID)
	if info.Username != "" {
		s.logger.Info("MAX bot link", "url", "https://max.ru/"+info.Username)
	}
	for event := range s.bot.Events(ctx) {
		if err := s.handle(ctx, event); err != nil {
			s.logger.Error("handle bot event", "error", err, "user_id", event.UserID)
		}
	}
	return ctx.Err()
}

func (s *Service) handle(ctx context.Context, event domain.Event) error {
	if event.Kind == domain.EventCallback && event.MessageID != "" {
		s.setRenderTarget(event.UserID, renderTarget{chatID: event.ChatID, messageID: event.MessageID, userID: event.UserID})
	}
	if target, ok := s.renderTarget(event.UserID); ok && target.chatID == event.ChatID {
		ctx = context.WithValue(ctx, renderTargetContextKey{}, target)
		if event.Kind == domain.EventMessage {
			ctx = context.WithValue(ctx, replaceRenderTargetContextKey{}, true)
		}
	}
	if event.Kind == domain.EventCallback {
		_ = s.bot.AnswerCallback(ctx, event.CallbackID, "Обрабатываю")
		return s.handleCallback(ctx, event)
	}

	user, registered := s.repo.UserByMaxID(event.UserID)
	if !registered {
		if s.hasSession(event.UserID) {
			return s.handleRegistrationInput(ctx, event)
		}
		return s.showRegistration(ctx, event.ChatID)
	}
	if event.DisplayName != "" && user.DisplayName != event.DisplayName {
		user.DisplayName = event.DisplayName
		s.repo.SaveUser(user)
		s.persistUser(ctx, user)
	}
	if !user.IsActive() {
		return s.send(ctx, event.ChatID, "⛔ Доступ заблокирован администратором.", nil)
	}
	if s.hasSession(event.UserID) {
		return s.handleSessionInput(ctx, event, user)
	}
	if strings.HasPrefix(strings.TrimSpace(event.Text), "/") {
		return s.sendHome(ctx, event.ChatID, user, "Используйте меню ниже — команды больше не нужны.")
	}
	return s.sendHome(ctx, event.ChatID, user, "Выберите действие в меню ниже.")
}

func (s *Service) setRenderTarget(userID int64, target renderTarget) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	s.targets[userID] = target
}

func (s *Service) renderTarget(userID int64) (renderTarget, bool) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	target, ok := s.targets[userID]
	return target, ok
}

func (s *Service) setSession(userID int64, session Session) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	s.sessions[userID] = session
}

func (s *Service) session(userID int64) (Session, bool) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	session, ok := s.sessions[userID]
	return session, ok
}

func (s *Service) hasSession(userID int64) bool {
	_, ok := s.session(userID)
	return ok
}

func (s *Service) clearSession(userID int64) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	delete(s.sessions, userID)
}

func (s *Service) send(ctx context.Context, chat int64, text string, buttons []domain.Button) error {
	outgoing := domain.OutgoingMessage{ChatID: chat, Text: text, Buttons: buttons}
	target, hasTarget := ctx.Value(renderTargetContextKey{}).(renderTarget)
	replaceTarget, _ := ctx.Value(replaceRenderTargetContextKey{}).(bool)
	if hasTarget && target.chatID == chat {
		if len(outgoing.Buttons) == 0 {
			if user, exists := s.repo.UserByMaxID(target.userID); exists {
				outgoing.Buttons = menuForUser(user)
			}
		}
		if replaceTarget {
			if err := s.bot.DeleteMessage(ctx, target.messageID); err != nil {
				s.logger.Warn("delete previous menu message", "error", err, "message_id", target.messageID)
			}
		} else {
			outgoing.MessageID = target.messageID
		}
	}
	messageID, err := s.bot.Send(ctx, outgoing)
	if err == nil && replaceTarget && messageID != "" {
		s.setRenderTarget(target.userID, renderTarget{chatID: chat, messageID: messageID, userID: target.userID})
	}
	return err
}

func (s *Service) sendNotification(ctx context.Context, userID int64, text string, menu []domain.Button, photos []domain.Photo) error {
	message := domain.OutgoingMessage{UserID: userID, Text: text, Photos: photos}
	if _, err := s.bot.Send(ctx, message); err != nil {
		if len(photos) == 0 {
			return err
		}
		message.Photos = nil
		if _, fallbackErr := s.bot.Send(ctx, message); fallbackErr != nil {
			return fmt.Errorf("send notification: %w; fallback: %v", err, fallbackErr)
		}
	}
	if len(menu) == 0 {
		return nil
	}
	_, err := s.bot.Send(ctx, domain.OutgoingMessage{UserID: userID, Text: "🏠 Главное меню", Buttons: menu})
	return err
}

func (s *Service) notifyUser(ctx context.Context, userID int64, kind domain.NotificationKind, taskID, text string, menu []domain.Button, photos ...[]domain.Photo) error {
	var attachments []domain.Photo
	if len(photos) > 0 {
		attachments = photos[0]
	}
	if s.notifications == nil {
		return s.sendNotification(ctx, userID, text, menu, attachments)
	}
	notification := domain.Notification{ID: newCode(), Kind: kind, RecipientUserID: userID, TaskID: taskID, Text: text, Buttons: menu, Photos: attachments}
	if err := s.notifications.Publish(ctx, notification); err != nil {
		s.logger.Warn("publish notification, using direct delivery", "error", err, "notification_id", notification.ID, "kind", kind, "user_id", userID)
		return s.sendNotification(ctx, userID, text, menu, attachments)
	}
	return nil
}

func (s *Service) sendHome(ctx context.Context, chat int64, user domain.User, prefix string) error {
	return s.send(ctx, chat, prefix+"\n\n🏠 Главное меню", menuForUser(user))
}

func roleNames(roles []domain.Role) []string {
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, role.Label())
	}
	return names
}
