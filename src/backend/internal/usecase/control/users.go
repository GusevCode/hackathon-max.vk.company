package control

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/access"
)

func (s *Service) listUsers(ctx context.Context, event domain.Event, actor domain.User) error {
	if !access.Can(actor, access.ActionViewUsers) {
		return s.sendHome(ctx, event.ChatID, actor, "Недостаточно прав.")
	}
	users := s.repo.Users(actor.OrganizationID)
	lines := []string{"👥 КОМАНДА\n\nВыберите пользователя для управления:"}
	buttons := make([]domain.Button, 0, len(users)+1)
	for _, user := range users {
		if actor.HasRole(domain.RoleManager) && !actor.HasRole(domain.RoleOperator) && !actor.HasRole(domain.RoleAdmin) && user.ManagerID != actor.ID && user.ID != actor.ID {
			continue
		}
		lines = append(lines, fmt.Sprintf("• %s — %s", userLabel(user), strings.Join(roleNames(user.Roles), ", ")))
		buttons = append(buttons, domain.Button{Text: roleEmoji(user) + " " + userLabel(user), Payload: "user:view:" + strconv.FormatInt(user.MaxUserID, 10), Row: len(buttons) / 2})
	}
	buttons = append(buttons, domain.Button{Text: "↩️ Главное меню", Payload: "menu:home", Row: len(buttons)/2 + 1})
	return s.send(ctx, event.ChatID, strings.Join(lines, "\n"), buttons)
}

func (s *Service) handleUserCallback(ctx context.Context, event domain.Event, actor domain.User, action, rawID string) error {
	targetID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return s.listUsers(ctx, event, actor)
	}
	target, ok := s.repo.UserByMaxID(targetID)
	if !ok || target.OrganizationID != actor.OrganizationID {
		return s.sendHome(ctx, event.ChatID, actor, "Пользователь не найден.")
	}
	switch action {
	case "view":
		return s.userCard(ctx, event, actor, target)
	case "block", "unblock":
		if !access.CanManageUser(actor, target) {
			return s.sendHome(ctx, event.ChatID, actor, "Недостаточно прав для изменения этого пользователя.")
		}
		target.Status = domain.UserActive
		if action == "block" {
			target.Status = domain.UserBlocked
		}
		s.repo.SaveUser(target)
		s.persistUser(ctx, target)
		return s.userCard(ctx, event, actor, target)
	case "assign":
		return s.managerChoices(ctx, event, actor, target)
	case "roles":
		return s.roleChoices(ctx, event, actor, target)
	default:
		return s.listUsers(ctx, event, actor)
	}
}

func (s *Service) userCard(ctx context.Context, event domain.Event, actor, target domain.User) error {
	if !access.CanViewUser(actor, target) {
		return s.sendHome(ctx, event.ChatID, actor, "Недостаточно прав для просмотра пользователя.")
	}
	text := fmt.Sprintf("👤 %s\n\nРоли: %s\nСтатус: %s", userLabel(target), strings.Join(roleNames(target.Roles), ", "), userStatusLabel(target.Status))
	buttons := []domain.Button{{Text: "↩️ К команде", Payload: "menu:users", Row: 2}}
	if access.CanManageUser(actor, target) {
		action := "block"
		label := "⛔ Заблокировать"
		if target.Status == domain.UserBlocked {
			action = "unblock"
			label = "✅ Разблокировать"
		}
		buttons = append([]domain.Button{{Text: label, Payload: "user:" + action + ":" + fmt.Sprint(target.MaxUserID), Row: 0}}, buttons...)
		if target.HasRole(domain.RoleEmployee) {
			buttons = append([]domain.Button{{Text: "🧑‍💼 Назначить руководителя", Payload: "user:assign:" + fmt.Sprint(target.MaxUserID), Row: 1}}, buttons...)
		}
		if access.Can(actor, access.ActionManageRoles) {
			buttons = append([]domain.Button{{Text: "🔐 Управление ролями", Payload: "user:roles:" + fmt.Sprint(target.MaxUserID), Row: 1}}, buttons...)
		}
	}
	return s.send(ctx, event.ChatID, text, buttons)
}

func (s *Service) roleChoices(ctx context.Context, event domain.Event, actor, target domain.User) error {
	if !access.Can(actor, access.ActionManageRoles) {
		return s.sendHome(ctx, event.ChatID, actor, "Только администратор управляет ролями.")
	}
	roles := []domain.Role{domain.RoleOperator, domain.RoleManager, domain.RoleEmployee}
	buttons := make([]domain.Button, 0, len(roles)+1)
	for _, role := range roles {
		if target.HasRole(role) {
			buttons = append(buttons, domain.Button{Text: "➖ Убрать: " + role.Label(), Payload: "user:role_remove:" + fmt.Sprint(target.MaxUserID) + ":" + string(role), Row: len(buttons)})
		} else {
			buttons = append(buttons, domain.Button{Text: "➕ Добавить: " + role.Label(), Payload: "user:role:" + fmt.Sprint(target.MaxUserID) + ":" + string(role), Row: len(buttons)})
		}
	}
	buttons = append(buttons, domain.Button{Text: "↩️ Назад", Payload: "user:view:" + fmt.Sprint(target.MaxUserID), Row: len(buttons)})
	return s.send(ctx, event.ChatID, "🔐 Управление ролями\n\n➕ добавляет роль, ➖ убирает роль. У пользователя должна остаться хотя бы одна роль.", buttons)
}

func (s *Service) removeUserRole(ctx context.Context, event domain.Event, actor domain.User, rawID string, role domain.Role) error {
	if !access.Can(actor, access.ActionManageRoles) || !role.Valid() || role == domain.RoleAdmin {
		return s.sendHome(ctx, event.ChatID, actor, "Недостаточно прав.")
	}
	targetID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return s.sendHome(ctx, event.ChatID, actor, "Пользователь не найден.")
	}
	target, ok := s.repo.UserByMaxID(targetID)
	if !ok || target.OrganizationID != actor.OrganizationID {
		return s.sendHome(ctx, event.ChatID, actor, "Пользователь не найден.")
	}
	if !target.HasRole(role) {
		return s.roleChoices(ctx, event, actor, target)
	}
	if len(target.Roles) == 1 {
		return s.send(ctx, event.ChatID, "Нельзя убрать последнюю роль пользователя. Сначала добавьте ему другую роль.", []domain.Button{{
			Text: "↩️ К ролям", Payload: "user:roles:" + fmt.Sprint(target.MaxUserID), Row: 0,
		}})
	}

	roles := make([]domain.Role, 0, len(target.Roles)-1)
	for _, assignedRole := range target.Roles {
		if assignedRole != role {
			roles = append(roles, assignedRole)
		}
	}
	target.Roles = roles
	s.repo.SaveUser(target)
	s.persistUser(ctx, target)
	return s.roleChoices(ctx, event, actor, target)
}

func (s *Service) addUserRole(ctx context.Context, event domain.Event, actor domain.User, rawID string, role domain.Role) error {
	if !access.Can(actor, access.ActionManageRoles) || !role.Valid() || role == domain.RoleAdmin {
		return s.sendHome(ctx, event.ChatID, actor, "Недостаточно прав.")
	}
	targetID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return s.sendHome(ctx, event.ChatID, actor, "Пользователь не найден.")
	}
	target, ok := s.repo.UserByMaxID(targetID)
	if !ok || target.OrganizationID != actor.OrganizationID {
		return s.sendHome(ctx, event.ChatID, actor, "Пользователь не найден.")
	}
	if !target.HasRole(role) {
		target.Roles = append(target.Roles, role)
		s.repo.SaveUser(target)
		s.persistUser(ctx, target)
	}
	return s.userCard(ctx, event, actor, target)
}

func (s *Service) managerChoices(ctx context.Context, event domain.Event, actor, target domain.User) error {
	if !access.CanManageUser(actor, target) || !target.HasRole(domain.RoleEmployee) {
		return s.sendHome(ctx, event.ChatID, actor, "Назначать руководителя можно только сотруднику.")
	}
	buttons := make([]domain.Button, 0)
	for _, manager := range s.repo.Users(actor.OrganizationID) {
		if manager.HasRole(domain.RoleManager) || manager.HasRole(domain.RoleOperator) || manager.HasRole(domain.RoleAdmin) {
			buttons = append(buttons, domain.Button{Text: "🧑‍💼 " + userLabel(manager), Payload: "user:manager:" + fmt.Sprint(target.MaxUserID) + ":" + fmt.Sprint(manager.MaxUserID), Row: len(buttons) / 2})
		}
	}
	buttons = append(buttons, domain.Button{Text: "↩️ Назад", Payload: "user:view:" + fmt.Sprint(target.MaxUserID), Row: len(buttons)/2 + 1})
	return s.send(ctx, event.ChatID, "Выберите руководителя для сотрудника:", buttons)
}

func (s *Service) assignManager(ctx context.Context, event domain.Event, actor domain.User, employeeID, managerID string) error {
	if !access.Can(actor, access.ActionAssignManager) {
		return s.sendHome(ctx, event.ChatID, actor, "Недостаточно прав.")
	}
	employeeMaxID, employeeErr := strconv.ParseInt(employeeID, 10, 64)
	managerMaxID, managerErr := strconv.ParseInt(managerID, 10, 64)
	if employeeErr != nil || managerErr != nil {
		return s.sendHome(ctx, event.ChatID, actor, "Пользователь не найден.")
	}
	employee, employeeOK := s.repo.UserByMaxID(employeeMaxID)
	manager, managerOK := s.repo.UserByMaxID(managerMaxID)
	if !employeeOK || !managerOK {
		return s.sendHome(ctx, event.ChatID, actor, "Проверьте роли пользователей.")
	}
	if employee.OrganizationID != actor.OrganizationID || manager.OrganizationID != actor.OrganizationID {
		return s.sendHome(ctx, event.ChatID, actor, "Проверьте роли пользователей.")
	}
	if !employee.HasRole(domain.RoleEmployee) {
		return s.sendHome(ctx, event.ChatID, actor, "Выбранный пользователь не является сотрудником.")
	}
	if !manager.HasRole(domain.RoleManager) && !manager.HasRole(domain.RoleOperator) && !manager.HasRole(domain.RoleAdmin) {
		return s.sendHome(ctx, event.ChatID, actor, "Проверьте роли пользователей.")
	}
	employee.ManagerID = manager.ID
	s.repo.SaveUser(employee)
	s.persistUser(ctx, employee)
	return s.userCard(ctx, event, actor, employee)
}

func roleEmoji(user domain.User) string {
	if user.HasRole(domain.RoleAdmin) {
		return "🛡"
	}
	if user.HasRole(domain.RoleOperator) {
		return "🏢"
	}
	if user.HasRole(domain.RoleManager) {
		return "🧑‍💼"
	}
	return "👷"
}

func userStatusLabel(status domain.UserStatus) string {
	if status == domain.UserBlocked {
		return "заблокирован"
	}
	return "активен"
}

func shortName(name string) string {
	if len([]rune(name)) > 18 {
		return string([]rune(name)[:18]) + "…"
	}
	return name
}

func userLabel(user domain.User) string {
	name := strings.TrimSpace(user.DisplayName)
	if name == "" {
		name = "Пользователь"
	}
	return fmt.Sprintf("%s (%d)", shortName(name), user.MaxUserID)
}
