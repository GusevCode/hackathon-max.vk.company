package control

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type Service struct {
	bot     BotGateway
	repo    Repository
	photos  PhotoStore
	storage Storage
	logger  *slog.Logger
	drafts  map[int64]string
}

func NewService(bot BotGateway, repo Repository, photos PhotoStore, logger *slog.Logger, storages ...Storage) *Service {
	var storage Storage
	if len(storages) > 0 {
		storage = storages[0]
	}
	return &Service{bot: bot, repo: repo, photos: photos, storage: storage, logger: logger, drafts: map[int64]string{}}
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
		return s.callback(ctx, event)
	}
	text := strings.TrimSpace(event.Text)
	if text == "" && len(event.Photos) == 0 {
		return nil
	}
	user, ok := s.repo.UserByMaxID(event.UserID)
	if !ok {
		if strings.HasPrefix(strings.ToUpper(text), "/JOIN ") {
			return s.join(ctx, event, strings.TrimSpace(text[6:]))
		}
		return s.send(ctx, event.ChatID, "Вы ещё не зарегистрированы. Получите одноразовый код приглашения и отправьте: /join КОД", nil)
	}
	if user.Status == domain.UserBlocked {
		return s.send(ctx, event.ChatID, "Доступ заблокирован администратором.", nil)
	}
	if len(event.Photos) > 0 {
		if strings.HasPrefix(strings.ToLower(text), "/before ") {
			return s.attachBefore(ctx, event, user)
		}
		if strings.HasPrefix(strings.ToLower(text), "/submit ") {
			return s.submit(ctx, event, user)
		}
	}
	if strings.HasPrefix(text, "/") {
		return s.command(ctx, event, user)
	}
	if draft := s.drafts[event.UserID]; draft != "" {
		return s.send(ctx, event.ChatID, "Ожидается команда. Используйте /help.", nil)
	}
	return s.send(ctx, event.ChatID, "Команда не распознана. Используйте /help.", nil)
}

func (s *Service) command(ctx context.Context, e domain.Event, u domain.User) error {
	parts := strings.SplitN(strings.TrimSpace(e.Text), " ", 2)
	cmd := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}
	switch cmd {
	case "/start":
		return s.send(ctx, e.ChatID, "🏠 Добро пожаловать в «ЖКХ Контроль»!\nВыберите нужное действие в меню ниже.", menuForUser(u))
	case "/help":
		return s.send(ctx, e.ChatID, helpText(u), append(menuForUser(u), domain.Button{Text: "↩️ Главное меню", Payload: "menu:home", Row: 99}))
	case "/admin":
		if !hasRole(u, domain.RoleAdmin) {
			return s.send(ctx, e.ChatID, "Команда доступна только администратору.", nil)
		}
		return s.adminMenu(ctx, e.ChatID)
	case "/invite":
		return s.invite(ctx, e, u, arg)
	case "/join":
		return s.join(ctx, e, arg)
	case "/users":
		return s.listUsers(ctx, e, u)
	case "/block":
		return s.block(ctx, e, u, arg, true)
	case "/unblock":
		return s.block(ctx, e, u, arg, false)
	case "/assign":
		return s.assign(ctx, e, u, arg)
	case "/role":
		return s.addRole(ctx, e, u, arg)
	case "/object":
		return s.createObject(ctx, e, u, arg)
	case "/worktype":
		return s.createWorkType(ctx, e, u, arg)
	case "/task":
		return s.createTask(ctx, e, u, arg)
	case "/tasks":
		return s.listTasks(ctx, e, u)
	case "/take":
		return s.takeTask(ctx, e, u, arg)
	case "/submit":
		return s.submit(ctx, e, u)
	case "/unable":
		return s.unable(ctx, e, u, arg)
	case "/accept":
		return s.review(ctx, e, u, arg, "accepted")
	case "/rework":
		return s.review(ctx, e, u, arg, "rework")
	default:
		return s.send(ctx, e.ChatID, "Неизвестная команда. Используйте /help.", nil)
	}
}

func helpText(u domain.User) string {
	base := "Доступные команды:\n/start\n/help\n/join КОД\n/tasks — мои задания\n/take ID — взять в работу\n/before ID — отправить фото до начала\n/submit ID | комментарий — отправить фото после выполнения\n/unable ID | причина — сообщить о невозможности"
	if hasRole(u, domain.RoleManager) || hasRole(u, domain.RoleAdmin) {
		base += "\n/task Название | описание | MAX_ID исполнителя | YYYY-MM-DD | OBJECT_ID | WORKTYPE_ID\n/accept ID\n/rework ID | комментарий\n/object название | адрес\n/worktype название"
	}
	if hasRole(u, domain.RoleAdmin) {
		base += "\n/admin\n/invite manager|employee\n/users\n/block MAX_ID\n/assign MAX_ID_исполнителя MAX_ID_руководителя"
		base += "\n/role MAX_ID admin|manager|employee"
	}
	return base
}

func menuForUser(u domain.User) []domain.Button {
	buttons := []domain.Button{
		{Text: "📋 Мои задания", Payload: "menu:tasks", Row: 0},
		{Text: "❓ Помощь", Payload: "menu:help", Row: 0},
	}
	if hasRole(u, domain.RoleManager) || hasRole(u, domain.RoleAdmin) {
		buttons = append(buttons,
			domain.Button{Text: "➕ Создать задание", Payload: "menu:create_task", Row: 1},
			domain.Button{Text: "👥 Исполнители", Payload: "menu:users", Row: 1},
		)
	}
	if hasRole(u, domain.RoleAdmin) {
		buttons = append(buttons,
			domain.Button{Text: "⚙️ Администрирование", Payload: "menu:admin", Row: 2},
			domain.Button{Text: "🏢 Объекты", Payload: "menu:objects", Row: 2},
			domain.Button{Text: "🧹 Виды работ", Payload: "menu:worktypes", Row: 2},
		)
	}
	return buttons
}

func (s *Service) adminMenu(ctx context.Context, chat int64) error {
	return s.send(ctx, chat, "⚙️ Панель администратора\n\nУправляйте доступом и справочниками организации.", []domain.Button{
		{Text: "👤 Пригласить руководителя", Payload: "admin:invite:manager", Row: 0},
		{Text: "👷 Пригласить исполнителя", Payload: "admin:invite:employee", Row: 0},
		{Text: "👥 Пользователи", Payload: "admin:users", Row: 1},
		{Text: "🏠 Новый объект", Payload: "menu:objects", Row: 1},
		{Text: "↩️ Главное меню", Payload: "menu:home", Row: 2},
	})
}
func (s *Service) invite(ctx context.Context, e domain.Event, u domain.User, arg string) error {
	if !hasRole(u, domain.RoleAdmin) {
		return s.send(ctx, e.ChatID, "Недостаточно прав.", nil)
	}
	fields := strings.Fields(arg)
	if len(fields) == 0 {
		return s.send(ctx, e.ChatID, "Формат: /invite manager или /invite employee", nil)
	}
	role := strings.ToLower(fields[0])
	if role != "manager" && role != "employee" {
		return s.send(ctx, e.ChatID, "Формат: /invite manager или /invite employee", nil)
	}
	code := newCode()
	invite := domain.Invite{Code: code, OrganizationID: u.OrganizationID, Roles: []domain.Role{domain.Role(role)}, ExpiresAt: time.Now().Add(24 * time.Hour)}
	s.repo.SaveInvite(invite)
	if s.storage != nil {
		if err := s.storage.SaveInvite(ctx, invite); err != nil {
			s.logger.Warn("persist invite", "error", err)
		}
	}
	return s.send(ctx, e.ChatID, fmt.Sprintf("Одноразовый код для роли %s: %s\nДействует 24 часа. Пользователь отправляет боту /join %s", role, code, code), nil)
}
func (s *Service) join(ctx context.Context, e domain.Event, code string) error {
	inv, ok := s.repo.Invite(strings.ToUpper(strings.TrimSpace(code)))
	if !ok || inv.UsedBy != 0 || time.Now().After(inv.ExpiresAt) {
		return s.send(ctx, e.ChatID, "Код приглашения недействителен или истёк.", nil)
	}
	u := domain.User{ID: fmt.Sprintf("user-%d", e.UserID), OrganizationID: inv.OrganizationID, MaxUserID: e.UserID, DisplayName: fmt.Sprintf("Пользователь %d", e.UserID), Roles: inv.Roles, Status: domain.UserActive, CreatedAt: time.Now()}
	s.repo.SaveUser(u)
	s.persistUser(ctx, u)
	inv.UsedBy = e.UserID
	s.repo.SaveInvite(inv)
	if s.storage != nil {
		if err := s.storage.SaveInvite(ctx, inv); err != nil {
			s.logger.Warn("persist invite", "error", err)
		}
	}
	return s.send(ctx, e.ChatID, "Регистрация завершена. Используйте /help.", nil)
}
func (s *Service) listUsers(ctx context.Context, e domain.Event, u domain.User) error {
	if !hasRole(u, domain.RoleAdmin) {
		return s.send(ctx, e.ChatID, "Недостаточно прав.", nil)
	}
	users := s.repo.Users(u.OrganizationID)
	lines := []string{"Пользователи организации:"}
	for _, v := range users {
		lines = append(lines, fmt.Sprintf("%d — %s — %s — %s", v.MaxUserID, v.DisplayName, strings.Join(roleNames(v.Roles), ","), v.Status))
	}
	return s.send(ctx, e.ChatID, strings.Join(lines, "\n"), nil)
}
func (s *Service) block(ctx context.Context, e domain.Event, u domain.User, arg string, blocked bool) error {
	if !hasRole(u, domain.RoleAdmin) {
		return s.send(ctx, e.ChatID, "Недостаточно прав.", nil)
	}
	id, _ := strconv.ParseInt(strings.TrimSpace(arg), 10, 64)
	v, ok := s.repo.UserByMaxID(id)
	if !ok || v.OrganizationID != u.OrganizationID {
		return s.send(ctx, e.ChatID, "Пользователь не найден.", nil)
	}
	if blocked {
		v.Status = domain.UserBlocked
	} else {
		v.Status = domain.UserActive
	}
	s.repo.SaveUser(v)
	s.persistUser(ctx, v)
	return s.send(ctx, e.ChatID, "Статус пользователя изменён.", nil)
}
func (s *Service) assign(ctx context.Context, e domain.Event, u domain.User, arg string) error {
	if !hasRole(u, domain.RoleAdmin) {
		return s.send(ctx, e.ChatID, "Недостаточно прав.", nil)
	}
	p := strings.Fields(arg)
	if len(p) != 2 {
		return s.send(ctx, e.ChatID, "Формат: /assign MAX_ID_исполнителя MAX_ID_руководителя", nil)
	}
	eid, _ := strconv.ParseInt(p[0], 10, 64)
	mid, _ := strconv.ParseInt(p[1], 10, 64)
	employee, eok := s.repo.UserByMaxID(eid)
	manager, mok := s.repo.UserByMaxID(mid)
	if !eok || !mok || employee.OrganizationID != u.OrganizationID || manager.OrganizationID != u.OrganizationID || !hasRole(manager, domain.RoleManager) {
		return s.send(ctx, e.ChatID, "Проверьте пользователей и роль руководителя.", nil)
	}
	employee.ManagerID = manager.ID
	s.repo.SaveUser(employee)
	s.persistUser(ctx, employee)
	return s.send(ctx, e.ChatID, "Руководитель назначен.", nil)
}

func (s *Service) addRole(ctx context.Context, e domain.Event, u domain.User, arg string) error {
	if !hasRole(u, domain.RoleAdmin) {
		return s.send(ctx, e.ChatID, "Недостаточно прав.", nil)
	}
	p := strings.Fields(arg)
	if len(p) != 2 {
		return s.send(ctx, e.ChatID, "Формат: /role MAX_ID admin|manager|employee", nil)
	}
	id, _ := strconv.ParseInt(p[0], 10, 64)
	role := domain.Role(strings.ToLower(p[1]))
	if role != domain.RoleAdmin && role != domain.RoleManager && role != domain.RoleEmployee {
		return s.send(ctx, e.ChatID, "Неизвестная роль.", nil)
	}
	target, ok := s.repo.UserByMaxID(id)
	if !ok || target.OrganizationID != u.OrganizationID {
		return s.send(ctx, e.ChatID, "Пользователь не найден.", nil)
	}
	if !hasRole(target, role) {
		target.Roles = append(target.Roles, role)
	}
	s.repo.SaveUser(target)
	s.persistUser(ctx, target)
	return s.send(ctx, e.ChatID, "Роль добавлена.", nil)
}
func (s *Service) createObject(ctx context.Context, e domain.Event, u domain.User, arg string) error {
	if !hasRole(u, domain.RoleAdmin) {
		return s.send(ctx, e.ChatID, "Только администратор может добавлять объекты.", nil)
	}
	p := strings.SplitN(arg, "|", 2)
	if len(p) != 2 {
		return s.send(ctx, e.ChatID, "Формат: /object название | адрес", nil)
	}
	id := newCode()
	object := domain.Object{ID: id, OrganizationID: u.OrganizationID, Name: strings.TrimSpace(p[0]), Address: strings.TrimSpace(p[1]), Kind: "housing"}
	s.repo.SaveObject(object)
	if s.storage != nil {
		if err := s.storage.SaveObject(ctx, object); err != nil {
			s.logger.Warn("persist object", "error", err, "object_id", id)
		}
	}
	return s.send(ctx, e.ChatID, "Объект создан, ID: "+id, nil)
}
func (s *Service) createWorkType(ctx context.Context, e domain.Event, u domain.User, arg string) error {
	if !hasRole(u, domain.RoleAdmin) {
		return s.send(ctx, e.ChatID, "Только администратор может добавлять виды работ.", nil)
	}
	if strings.TrimSpace(arg) == "" {
		return s.send(ctx, e.ChatID, "Формат: /worktype название", nil)
	}
	id := newCode()
	workType := domain.WorkType{ID: id, OrganizationID: u.OrganizationID, Name: strings.TrimSpace(arg)}
	s.repo.SaveWorkType(workType)
	if s.storage != nil {
		if err := s.storage.SaveWorkType(ctx, workType); err != nil {
			s.logger.Warn("persist work type", "error", err, "work_type_id", id)
		}
	}
	return s.send(ctx, e.ChatID, "Вид работы создан, ID: "+id, nil)
}
func (s *Service) createTask(ctx context.Context, e domain.Event, u domain.User, arg string) error {
	if !hasRole(u, domain.RoleManager) && !hasRole(u, domain.RoleAdmin) {
		return s.send(ctx, e.ChatID, "Только руководитель или администратор может создавать задания.", nil)
	}
	p := strings.Split(arg, "|")
	if len(p) < 4 {
		return s.send(ctx, e.ChatID, "Формат: /task название | описание | MAX_ID исполнителя | YYYY-MM-DD | OBJECT_ID | WORKTYPE_ID", nil)
	}
	eid, _ := strconv.ParseInt(strings.TrimSpace(p[2]), 10, 64)
	employee, ok := s.repo.UserByMaxID(eid)
	if !ok || employee.OrganizationID != u.OrganizationID || !hasRole(employee, domain.RoleEmployee) {
		return s.send(ctx, e.ChatID, "Исполнитель не найден в вашей организации.", nil)
	}
	if hasRole(u, domain.RoleManager) && !hasRole(u, domain.RoleAdmin) && employee.ManagerID != u.ID {
		return s.send(ctx, e.ChatID, "Исполнитель не закреплён за вами.", nil)
	}
	due, _ := time.Parse("2006-01-02", strings.TrimSpace(p[3]))
	objectID, workTypeID := "", ""
	if len(p) > 4 {
		objectID = strings.TrimSpace(p[4])
	}
	if len(p) > 5 {
		workTypeID = strings.TrimSpace(p[5])
	}
	if objectID != "" {
		found := false
		for _, object := range s.repo.Objects(u.OrganizationID) {
			if object.ID == objectID {
				found = true
				break
			}
		}
		if !found {
			return s.send(ctx, e.ChatID, "Объект не найден в вашей организации.", nil)
		}
	}
	if workTypeID != "" {
		found := false
		for _, workType := range s.repo.WorkTypes(u.OrganizationID) {
			if workType.ID == workTypeID {
				found = true
				break
			}
		}
		if !found {
			return s.send(ctx, e.ChatID, "Вид работы не найден в вашей организации.", nil)
		}
	}
	id := newCode()
	t := domain.Task{ID: id, OrganizationID: u.OrganizationID, Title: strings.TrimSpace(p[0]), Description: strings.TrimSpace(p[1]), ObjectID: objectID, WorkTypeID: workTypeID, AssigneeID: employee.ID, ManagerID: u.ID, DueAt: due, Priority: domain.PriorityNormal, Status: domain.TaskAssigned, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	s.repo.SaveTask(t)
	s.persistTask(ctx, t)
	if err := s.send(ctx, employee.MaxUserID, fmt.Sprintf("Вам назначено задание %s: %s\nСрок: %s\nИспользуйте /tasks", t.ID, t.Title, t.DueAt.Format("02.01.2006")), nil); err != nil {
		s.logger.Warn("notify employee", "error", err, "task_id", t.ID)
	}
	return s.send(ctx, e.ChatID, "Задание создано, ID: "+id, nil)
}
func (s *Service) listTasks(ctx context.Context, e domain.Event, u domain.User) error {
	tasks := s.repo.Tasks(u.OrganizationID)
	lines := []string{"📋 Задания\nНажмите на задание, чтобы открыть карточку:"}
	buttons := make([]domain.Button, 0)
	for _, t := range tasks {
		if hasRole(u, domain.RoleEmployee) && t.AssigneeID != u.ID {
			continue
		}
		if hasRole(u, domain.RoleManager) && !hasRole(u, domain.RoleAdmin) && t.ManagerID != u.ID {
			continue
		}
		lines = append(lines, fmt.Sprintf("• %s — %s", statusLabel(t.Status), t.Title))
		buttons = append(buttons, domain.Button{Text: statusLabel(t.Status) + " " + shortID(t.ID), Payload: "task:view:" + t.ID, Row: len(buttons) / 2})
	}
	if len(buttons) == 0 {
		lines = append(lines, "\nПока заданий нет.")
	}
	buttons = append(buttons, domain.Button{Text: "↩️ Главное меню", Payload: "menu:home", Row: len(buttons)/2 + 1})
	return s.send(ctx, e.ChatID, strings.Join(lines, "\n"), buttons)
}

func statusLabel(status domain.TaskStatus) string {
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

func shortID(id string) string {
	if len(id) > 6 {
		return id[:6]
	}
	return id
}
func (s *Service) takeTask(ctx context.Context, e domain.Event, u domain.User, arg string) error {
	t, ok := s.repo.Task(strings.TrimSpace(arg))
	if !ok || t.AssigneeID != u.ID {
		return s.send(ctx, e.ChatID, "Задание не найдено.", nil)
	}
	t.Status = domain.TaskInProgress
	t.UpdatedAt = time.Now()
	s.repo.SaveTask(t)
	s.persistTask(ctx, t)
	return s.send(ctx, e.ChatID, "Задание взято в работу. После выполнения отправьте фото с командой /submit "+t.ID+" | комментарий", nil)
}
func (s *Service) submit(ctx context.Context, e domain.Event, u domain.User) error {
	p := strings.SplitN(strings.TrimSpace(strings.TrimPrefix(e.Text, "/submit ")), "|", 2)
	if len(p) != 2 {
		return s.send(ctx, e.ChatID, "Нужны фото и комментарий: /submit ID | что сделано", nil)
	}
	t, ok := s.repo.Task(strings.TrimSpace(p[0]))
	if !ok || t.AssigneeID != u.ID {
		return s.send(ctx, e.ChatID, "Задание не найдено.", nil)
	}
	if len(e.Photos) == 0 {
		return s.send(ctx, e.ChatID, "Приложите хотя бы одну фотографию.", nil)
	}
	t.AfterPhotos = append(t.AfterPhotos, e.Photos...)
	if s.photos != nil {
		if err := s.persistPhotos(ctx, t.ID, "after", e.Photos); err != nil {
			return s.send(ctx, e.ChatID, "Не удалось сохранить фотографию. Повторите отправку отчёта.", nil)
		}
	}
	t.Comment = strings.TrimSpace(p[1])
	t.Status = domain.TaskSubmitted
	t.UpdatedAt = time.Now()
	s.repo.SaveTask(t)
	s.persistTask(ctx, t)
	if manager, ok := s.userByID(t.OrganizationID, t.ManagerID); ok {
		if err := s.send(ctx, manager.MaxUserID, fmt.Sprintf("Фотоотчёт по заданию %s ожидает проверки.\n%s", t.ID, t.Title), []domain.Button{{Text: "Принять", Payload: "task:accept:" + t.ID}, {Text: "На переделку", Payload: "task:rework:" + t.ID}}); err != nil {
			s.logger.Warn("notify manager", "error", err, "task_id", t.ID)
		}
	}
	return s.send(ctx, e.ChatID, "Фотоотчёт отправлен руководителю на проверку.", nil)
}

func (s *Service) attachBefore(ctx context.Context, e domain.Event, u domain.User) error {
	id := strings.TrimSpace(strings.TrimPrefix(e.Text, "/before "))
	t, ok := s.repo.Task(id)
	if !ok || t.AssigneeID != u.ID {
		return s.send(ctx, e.ChatID, "Задание не найдено.", nil)
	}
	t.BeforePhotos = append(t.BeforePhotos, e.Photos...)
	if s.photos != nil {
		if err := s.persistPhotos(ctx, t.ID, "before", e.Photos); err != nil {
			return s.send(ctx, e.ChatID, "Не удалось сохранить фотографию. Повторите отправку.", nil)
		}
	}
	t.Status = domain.TaskInProgress
	t.UpdatedAt = time.Now()
	s.repo.SaveTask(t)
	s.persistTask(ctx, t)
	return s.send(ctx, e.ChatID, "Фото до начала сохранены. После выполнения отправьте фото с командой /submit "+t.ID+" | комментарий", nil)
}
func (s *Service) unable(ctx context.Context, e domain.Event, u domain.User, arg string) error {
	p := strings.SplitN(arg, "|", 2)
	if len(p) != 2 {
		return s.send(ctx, e.ChatID, "Формат: /unable ID | причина", nil)
	}
	t, ok := s.repo.Task(strings.TrimSpace(p[0]))
	if !ok || t.AssigneeID != u.ID {
		return s.send(ctx, e.ChatID, "Задание не найдено.", nil)
	}
	t.Status = domain.TaskUnable
	t.Comment = strings.TrimSpace(p[1])
	s.repo.SaveTask(t)
	s.persistTask(ctx, t)
	return s.send(ctx, e.ChatID, "Причина сохранена и передана руководителю.", nil)
}
func (s *Service) review(ctx context.Context, e domain.Event, u domain.User, arg, decision string) error {
	p := strings.SplitN(arg, "|", 2)
	id := strings.TrimSpace(p[0])
	t, ok := s.repo.Task(id)
	if !ok || t.ManagerID != u.ID && !hasRole(u, domain.RoleAdmin) {
		return s.send(ctx, e.ChatID, "Задание не найдено или недоступно.", nil)
	}
	if decision == "rework" && (len(p) < 2 || strings.TrimSpace(p[1]) == "") {
		return s.send(ctx, e.ChatID, "При отправке на переделку комментарий обязателен: /rework ID | что исправить", nil)
	}
	t.Status = domain.TaskAccepted
	if decision == "rework" {
		t.Status = domain.TaskRework
		t.Comment = strings.TrimSpace(p[1])
	}
	s.repo.SaveTask(t)
	s.persistTask(ctx, t)
	review := domain.Review{TaskID: t.ID, ReviewerID: u.ID, Decision: decision, Comment: t.Comment, CreatedAt: time.Now()}
	s.repo.SaveReview(review)
	if s.storage != nil {
		if err := s.storage.SaveReview(ctx, review); err != nil {
			s.logger.Warn("persist review", "error", err, "task_id", t.ID)
		}
	}
	if employee, ok := s.userByID(t.OrganizationID, t.AssigneeID); ok {
		message := fmt.Sprintf("По заданию %s принято решение: %s", t.ID, decision)
		if decision == "rework" {
			message += "\nКомментарий: " + t.Comment
		}
		if err := s.send(ctx, employee.MaxUserID, message, nil); err != nil {
			s.logger.Warn("notify employee", "error", err, "task_id", t.ID)
		}
	}
	return s.send(ctx, e.ChatID, fmt.Sprintf("Решение по заданию %s сохранено: %s", t.ID, decision), nil)
}
func (s *Service) callback(ctx context.Context, e domain.Event) error {
	p := strings.Split(e.Payload, ":")
	if len(p) == 2 && p[0] == "menu" {
		u, ok := s.repo.UserByMaxID(e.UserID)
		if !ok {
			return nil
		}
		switch p[1] {
		case "home":
			return s.send(ctx, e.ChatID, "🏠 Главное меню\nВыберите нужное действие:", menuForUser(u))
		case "tasks":
			return s.listTasks(ctx, e, u)
		case "help":
			return s.send(ctx, e.ChatID, helpText(u), append(menuForUser(u), domain.Button{Text: "↩️ Главное меню", Payload: "menu:home", Row: 99}))
		case "admin":
			if hasRole(u, domain.RoleAdmin) {
				return s.adminMenu(ctx, e.ChatID)
			}
		case "users":
			if hasRole(u, domain.RoleAdmin) {
				return s.listUsers(ctx, e, u)
			}
			if hasRole(u, domain.RoleManager) {
				return s.listTeam(ctx, e, u)
			}
		case "create_task":
			if hasRole(u, domain.RoleManager) || hasRole(u, domain.RoleAdmin) {
				return s.send(ctx, e.ChatID, "➕ Создание задания\n\nФормат:\n/task Название | описание | MAX_ID исполнителя | YYYY-MM-DD | OBJECT_ID | WORKTYPE_ID\n\nПоля объекта и вида работы можно оставить пустыми.", []domain.Button{{Text: "📋 К заданиям", Payload: "menu:tasks", Row: 0}, {Text: "↩️ Главное меню", Payload: "menu:home", Row: 0}})
			}
		case "objects":
			if hasRole(u, domain.RoleAdmin) {
				return s.send(ctx, e.ChatID, "🏠 Объекты\n\nСоздание:\n/object Название объекта | Адрес", []domain.Button{{Text: "↩️ Главное меню", Payload: "menu:home", Row: 0}})
			}
		case "worktypes":
			if hasRole(u, domain.RoleAdmin) {
				return s.send(ctx, e.ChatID, "🧹 Виды работ\n\nСоздание:\n/worktype Название работы", []domain.Button{{Text: "↩️ Главное меню", Payload: "menu:home", Row: 0}})
			}
		}
		return s.send(ctx, e.ChatID, "Недостаточно прав для этого раздела.", menuForUser(u))
	}
	if len(p) == 3 && p[0] == "task" && p[1] == "view" {
		u, ok := s.repo.UserByMaxID(e.UserID)
		if ok {
			return s.taskCard(ctx, e, u, p[2])
		}
		return nil
	}
	if len(p) == 3 && p[0] == "task" {
		u, ok := s.repo.UserByMaxID(e.UserID)
		if !ok {
			return nil
		}
		if p[1] == "accept" {
			return s.review(ctx, e, u, p[2], "accepted")
		}
		if p[1] == "take" {
			return s.takeTask(ctx, e, u, p[2])
		}
		if p[1] == "before" {
			return s.send(ctx, e.ChatID, "📷 Прикрепите фотографию к сообщению с подписью:\n/before "+p[2], nil)
		}
		if p[1] == "submit" {
			return s.send(ctx, e.ChatID, "📸 Прикрепите фото после выполнения и подпишите сообщение:\n/submit "+p[2]+" | что сделано", nil)
		}
		if p[1] == "rework" {
			return s.send(ctx, e.ChatID, "Для переделки отправьте: /rework "+p[2]+" | что исправить", nil)
		}
	}
	if len(p) == 3 && p[0] == "admin" && p[1] == "users" {
		u, ok := s.repo.UserByMaxID(e.UserID)
		if ok {
			return s.listUsers(ctx, e, u)
		}
	}
	if len(p) == 3 && p[0] == "admin" && p[1] == "invite" {
		u, ok := s.repo.UserByMaxID(e.UserID)
		if ok {
			return s.invite(ctx, e, u, p[2])
		}
	}
	return nil
}

func (s *Service) listTeam(ctx context.Context, e domain.Event, u domain.User) error {
	lines := []string{"👥 Ваши исполнители:"}
	for _, employee := range s.repo.Users(u.OrganizationID) {
		if employee.ManagerID == u.ID && hasRole(employee, domain.RoleEmployee) {
			lines = append(lines, fmt.Sprintf("• %d — %s", employee.MaxUserID, employee.DisplayName))
		}
	}
	return s.send(ctx, e.ChatID, strings.Join(lines, "\n"), []domain.Button{{Text: "↩️ Главное меню", Payload: "menu:home", Row: 0}})
}

func (s *Service) taskCard(ctx context.Context, e domain.Event, u domain.User, id string) error {
	task, ok := s.repo.Task(id)
	if !ok || task.OrganizationID != u.OrganizationID {
		return s.send(ctx, e.ChatID, "Задание не найдено.", menuForUser(u))
	}
	allowed := hasRole(u, domain.RoleAdmin) || task.AssigneeID == u.ID || task.ManagerID == u.ID
	if !allowed {
		return s.send(ctx, e.ChatID, "У вас нет доступа к этому заданию.", menuForUser(u))
	}
	text := fmt.Sprintf("%s %s\n\n%s\n\nСтатус: %s\nСрок: %s\nФото: до — %d, после — %d", statusLabel(task.Status), task.Title, task.Description, task.Status, task.DueAt.Format("02.01.2006"), len(task.BeforePhotos), len(task.AfterPhotos))
	if task.Comment != "" {
		text += "\nКомментарий: " + task.Comment
	}
	buttons := make([]domain.Button, 0)
	if hasRole(u, domain.RoleEmployee) && task.AssigneeID == u.ID {
		if task.Status == domain.TaskAssigned || task.Status == domain.TaskRework {
			buttons = append(buttons, domain.Button{Text: "▶️ Взять в работу", Payload: "task:take:" + task.ID, Row: 0})
		}
		if task.Status == domain.TaskInProgress || task.Status == domain.TaskRework {
			buttons = append(buttons, domain.Button{Text: "📷 Фото до", Payload: "task:before:" + task.ID, Row: 0}, domain.Button{Text: "📸 Отправить отчёт", Payload: "task:submit:" + task.ID, Row: 1})
		}
	}
	if (hasRole(u, domain.RoleManager) || hasRole(u, domain.RoleAdmin)) && task.Status == domain.TaskSubmitted {
		buttons = append(buttons, domain.Button{Text: "✅ Принять", Payload: "task:accept:" + task.ID, Row: 0}, domain.Button{Text: "🔁 На переделку", Payload: "task:rework:" + task.ID, Row: 0})
	}
	buttons = append(buttons, domain.Button{Text: "↩️ К заданиям", Payload: "menu:tasks", Row: 2})
	return s.send(ctx, e.ChatID, text, buttons)
}
func (s *Service) send(ctx context.Context, chat int64, text string, buttons []domain.Button) error {
	return s.bot.Send(ctx, domain.OutgoingMessage{ChatID: chat, Text: text, Buttons: buttons})
}
func hasRole(u domain.User, r domain.Role) bool {
	for _, v := range u.Roles {
		if v == r {
			return true
		}
	}
	return false
}
func roleNames(rs []domain.Role) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = string(r)
	}
	return out
}

func (s *Service) userByID(organizationID, id string) (domain.User, bool) {
	for _, user := range s.repo.Users(organizationID) {
		if user.ID == id {
			return user, true
		}
	}
	return domain.User{}, false
}

func (s *Service) persistUser(ctx context.Context, user domain.User) {
	if s.storage != nil {
		if err := s.storage.SaveUser(ctx, user); err != nil {
			s.logger.Warn("persist user", "error", err, "user_id", user.MaxUserID)
		}
	}
}

func (s *Service) persistTask(ctx context.Context, task domain.Task) {
	if s.storage != nil {
		if err := s.storage.SaveTask(ctx, task); err != nil {
			s.logger.Warn("persist task", "error", err, "task_id", task.ID)
		}
	}
}

func (s *Service) persistPhotos(ctx context.Context, taskID, kind string, photos []domain.Photo) error {
	for i, photo := range photos {
		key := fmt.Sprintf("tasks/%s/%s/%d-%d.jpg", taskID, kind, time.Now().UnixNano(), i)
		if err := s.photos.UploadURL(ctx, key, photo.URL); err != nil {
			return err
		}
		if s.storage != nil {
			evidence := domain.Evidence{ID: newCode(), TaskID: taskID, Kind: kind, ObjectKey: key, CreatedAt: time.Now()}
			if err := s.storage.SaveEvidence(ctx, evidence); err != nil {
				return err
			}
		}
	}
	return nil
}
func newCode() string {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "LOCAL1"
	}
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]byte, 8)
	for i := range out {
		out[i] = alphabet[int(b[i%len(b)])%len(alphabet)]
	}
	return string(out)
}
