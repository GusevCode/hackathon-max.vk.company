package control

import (
	"context"
	"fmt"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/access"
)

func (s *Service) showRegistration(ctx context.Context, chatID int64) error {
	return s.send(ctx, chatID, "👋 Добро пожаловать в «ЖКХ Контроль»!\n\nЧтобы подключиться к организации, нажмите кнопку и введите пригласительный код.", []domain.Button{
		{Text: "🎟 Ввести код приглашения", Payload: "auth:invite", Row: 0},
	})
}

func menuForUser(user domain.User) []domain.Button {
	buttons := []domain.Button{
		{Text: "📋 Мои задания", Payload: "menu:tasks", Row: 0},
		{Text: "❓ Помощь", Payload: "menu:help", Row: 0},
	}
	if access.Can(user, access.ActionCreateTask) {
		buttons = append(buttons, domain.Button{Text: "➕ Новое задание", Payload: "menu:create_task", Row: 1})
	}
	if access.Can(user, access.ActionViewUsers) {
		buttons = append(buttons, domain.Button{Text: "👥 Команда", Payload: "menu:users", Row: 1})
	}
	if access.Can(user, access.ActionManageObjects) || access.Can(user, access.ActionManageWorkType) {
		buttons = append(buttons,
			domain.Button{Text: "🏢 Объекты", Payload: "menu:objects", Row: 2},
			domain.Button{Text: "🧹 Виды работ", Payload: "menu:worktypes", Row: 2},
		)
	}
	if user.HasRole(domain.RoleAdmin) || user.HasRole(domain.RoleOperator) {
		buttons = append(buttons, domain.Button{Text: "⚙️ Управление", Payload: "menu:admin", Row: 3})
	}
	return buttons
}

func (s *Service) adminMenu(ctx context.Context, chatID int64, user domain.User) error {
	buttons := []domain.Button{
		{Text: "🎟 Пригласить пользователя", Payload: "admin:invite", Row: 0},
		{Text: "👥 Пользователи", Payload: "admin:users", Row: 1},
	}
	if user.HasRole(domain.RoleAdmin) {
		buttons = append(buttons, domain.Button{Text: "🧹 Очистить задания и фото", Payload: "admin:clear", Row: 2})
	}
	buttons = append(buttons, domain.Button{Text: "↩️ Главное меню", Payload: "menu:home", Row: len(buttons)})
	label := "⚙️ УПРАВЛЕНИЕ ОРГАНИЗАЦИЕЙ"
	if access.Can(user, access.ActionManageRoles) {
		label = "🛡 АДМИНИСТРИРОВАНИЕ"
	}
	return s.send(ctx, chatID, fmt.Sprintf("%s\n\nДоступы, приглашения и пользователи организации.", label), buttons)
}

func (s *Service) inviteMenu(ctx context.Context, chatID int64, user domain.User) error {
	buttons := make([]domain.Button, 0, 4)
	if access.CanInviteRole(user, domain.RoleEmployee) {
		buttons = append(buttons, domain.Button{Text: "👷 Сотрудник", Payload: "admin:invite:employee", Row: 0})
	}
	if access.CanInviteRole(user, domain.RoleManager) {
		buttons = append(buttons, domain.Button{Text: "🧑‍💼 Руководитель", Payload: "admin:invite:manager", Row: 1})
	}
	if access.CanInviteRole(user, domain.RoleOperator) {
		buttons = append(buttons, domain.Button{Text: "🏢 Управляющий", Payload: "admin:invite:operator", Row: 2})
	}
	buttons = append(buttons, domain.Button{Text: "↩️ Назад", Payload: "menu:admin", Row: 3})
	return s.send(ctx, chatID, "🎟 СОЗДАНИЕ ПРИГЛАСИТЕЛЬНОГО КОДА\n\nВыберите роль нового пользователя:", buttons)
}

func helpText() string {
	return "❓ ПОМОЩЬ\n\nВсе действия выполняются кнопками.\n\n📋 Сотрудник получает задания, отмечает начало работы и отправляет фотоотчёт.\n🧑‍💼 Руководитель создаёт задания для своей команды и проверяет отчёты.\n🏢 Управляющий ведёт объекты, виды работ и операционный контур организации.\n🛡 Администратор управляет доступами и приглашениями."
}
