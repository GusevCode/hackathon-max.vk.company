package access

import "github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"

type Action string

const (
	ActionInviteEmployee Action = "invite_employee"
	ActionInviteManager  Action = "invite_manager"
	ActionInviteOperator Action = "invite_operator"
	ActionViewUsers      Action = "view_users"
	ActionManageUsers    Action = "manage_users"
	ActionAssignManager  Action = "assign_manager"
	ActionManageRoles    Action = "manage_roles"
	ActionManageObjects  Action = "manage_objects"
	ActionManageWorkType Action = "manage_work_types"
	ActionCreateTask     Action = "create_task"
	ActionEditTask       Action = "edit_task"
	ActionCloseTask      Action = "close_task"
	ActionViewAllTasks   Action = "view_all_tasks"
	ActionReviewTask     Action = "review_task"
	ActionExecuteTask    Action = "execute_task"
)

// Can answers only global role capability. Resource ownership and organization
// boundaries are checked by the corresponding application handler.
func Can(user domain.User, action Action) bool {
	if !user.IsActive() {
		return false
	}
	if user.HasRole(domain.RoleAdmin) {
		return true
	}
	switch action {
	case ActionInviteEmployee, ActionInviteManager, ActionManageUsers,
		ActionAssignManager, ActionViewAllTasks:
		return user.HasRole(domain.RoleOperator)
	case ActionManageObjects, ActionManageWorkType:
		return user.HasRole(domain.RoleOperator) || user.HasRole(domain.RoleManager)
	case ActionViewUsers:
		return user.HasRole(domain.RoleOperator) || user.HasRole(domain.RoleManager)
	case ActionCreateTask, ActionEditTask:
		return user.HasRole(domain.RoleOperator) || user.HasRole(domain.RoleManager)
	case ActionCloseTask:
		return user.HasRole(domain.RoleManager)
	case ActionReviewTask:
		return user.HasRole(domain.RoleOperator) || user.HasRole(domain.RoleManager)
	case ActionExecuteTask:
		return user.HasRole(domain.RoleEmployee)
	case ActionInviteOperator, ActionManageRoles:
		return false
	default:
		return false
	}
}

func CanInviteRole(user domain.User, role domain.Role) bool {
	switch role {
	case domain.RoleEmployee:
		return Can(user, ActionInviteEmployee)
	case domain.RoleManager:
		return Can(user, ActionInviteManager)
	case domain.RoleOperator:
		return Can(user, ActionInviteOperator)
	default:
		return false
	}
}

func CanReviewTask(user domain.User, task domain.Task) bool {
	if !Can(user, ActionReviewTask) || user.OrganizationID != task.OrganizationID {
		return false
	}
	return user.HasRole(domain.RoleAdmin) || user.HasRole(domain.RoleOperator) || task.ManagerID == user.ID
}

func CanViewTask(user domain.User, task domain.Task) bool {
	if user.OrganizationID != task.OrganizationID || !user.IsActive() || task.Status == domain.TaskClosed {
		return false
	}
	return user.HasRole(domain.RoleAdmin) || user.HasRole(domain.RoleOperator) || task.ManagerID == user.ID || task.AssigneeID == user.ID
}

func CanCloseTask(user domain.User, task domain.Task) bool {
	if !Can(user, ActionCloseTask) || user.OrganizationID != task.OrganizationID || task.Status == domain.TaskClosed {
		return false
	}
	return user.HasRole(domain.RoleAdmin) || (user.HasRole(domain.RoleManager) && task.ManagerID == user.ID)
}

func CanEditTask(user domain.User, task domain.Task) bool {
	if !Can(user, ActionEditTask) || user.OrganizationID != task.OrganizationID {
		return false
	}
	return user.HasRole(domain.RoleAdmin) || user.HasRole(domain.RoleOperator) || task.ManagerID == user.ID
}

func CanManageUser(actor, target domain.User) bool {
	if actor.OrganizationID != target.OrganizationID || !Can(actor, ActionManageUsers) {
		return false
	}
	if target.HasRole(domain.RoleAdmin) || target.HasRole(domain.RoleOperator) {
		return actor.HasRole(domain.RoleAdmin)
	}
	return true
}

func CanViewUser(actor, target domain.User) bool {
	if actor.OrganizationID != target.OrganizationID || !actor.IsActive() {
		return false
	}
	if actor.HasRole(domain.RoleAdmin) || actor.HasRole(domain.RoleOperator) || actor.ID == target.ID {
		return true
	}
	return actor.HasRole(domain.RoleManager) && target.ManagerID == actor.ID
}
