package control

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type Service struct {
	bot     BotGateway
	repo    Repository
	photos  PhotoStore
	storage Storage
	logger  *slog.Logger

	sessionsMu sync.Mutex
	sessions   map[int64]Session
}

type renderTargetContextKey struct{}

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
		sessions: make(map[int64]Session),
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
	if event.Kind == domain.EventCallback {
		_ = s.bot.AnswerCallback(ctx, event.CallbackID, "Обрабатываю")
		if event.MessageID != "" {
			ctx = context.WithValue(ctx, renderTargetContextKey{}, renderTarget{chatID: event.ChatID, messageID: event.MessageID, userID: event.UserID})
		}
		return s.handleCallback(ctx, event)
	}

	user, registered := s.repo.UserByMaxID(event.UserID)
	if !registered {
		if s.hasSession(event.UserID) {
			return s.handleRegistrationInput(ctx, event)
		}
		return s.showRegistration(ctx, event.ChatID)
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
	if target, ok := ctx.Value(renderTargetContextKey{}).(renderTarget); ok && target.chatID == chat {
		if len(outgoing.Buttons) == 0 {
			if user, exists := s.repo.UserByMaxID(target.userID); exists {
				outgoing.Buttons = menuForUser(user)
			}
		}
		outgoing.MessageID = target.messageID
	}
	return s.bot.Send(ctx, outgoing)
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
