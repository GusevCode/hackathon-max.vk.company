package access_test

import (
	"testing"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/access"
)

func TestRoleMatrix(t *testing.T) {
	cases := []struct {
		name   string
		role   domain.Role
		action access.Action
		want   bool
	}{
		{"admin invites operator", domain.RoleAdmin, access.ActionInviteOperator, true},
		{"operator invites manager", domain.RoleOperator, access.ActionInviteManager, true},
		{"operator cannot invite operator", domain.RoleOperator, access.ActionInviteOperator, false},
		{"manager creates task", domain.RoleManager, access.ActionCreateTask, true},
		{"manager edits task", domain.RoleManager, access.ActionEditTask, true},
		{"manager closes task", domain.RoleManager, access.ActionCloseTask, true},
		{"manager manages objects", domain.RoleManager, access.ActionManageObjects, true},
		{"manager manages work types", domain.RoleManager, access.ActionManageWorkType, true},
		{"manager reviews own task capability", domain.RoleManager, access.ActionReviewTask, true},
		{"employee executes task", domain.RoleEmployee, access.ActionExecuteTask, true},
		{"employee cannot manage users", domain.RoleEmployee, access.ActionManageUsers, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user := domain.User{OrganizationID: "org", Roles: []domain.Role{tc.role}, Status: domain.UserActive}
			if got := access.Can(user, tc.action); got != tc.want {
				t.Fatalf("Can(%s, %s) = %v, want %v", tc.role, tc.action, got, tc.want)
			}
		})
	}
}

func TestCanCloseTask(t *testing.T) {
	task := domain.Task{OrganizationID: "org", ManagerID: "manager", Status: domain.TaskInProgress}
	manager := domain.User{ID: "manager", OrganizationID: "org", Roles: []domain.Role{domain.RoleManager}, Status: domain.UserActive}
	otherManager := domain.User{ID: "other", OrganizationID: "org", Roles: []domain.Role{domain.RoleManager}, Status: domain.UserActive}
	employee := domain.User{ID: "employee", OrganizationID: "org", Roles: []domain.Role{domain.RoleEmployee}, Status: domain.UserActive}

	if !access.CanCloseTask(manager, task) {
		t.Fatal("assigned manager should be able to close task")
	}
	if access.CanCloseTask(otherManager, task) {
		t.Fatal("another manager should not be able to close task")
	}
	if access.CanCloseTask(employee, task) {
		t.Fatal("employee should not be able to close task")
	}
	operator := domain.User{ID: "operator", OrganizationID: "org", Roles: []domain.Role{domain.RoleOperator}, Status: domain.UserActive}
	if access.CanCloseTask(operator, task) {
		t.Fatal("operator should not be able to close task")
	}
	task.Status = domain.TaskClosed
	if access.CanViewTask(manager, task) || access.CanViewTask(employee, task) {
		t.Fatal("closed task should be hidden from active task views")
	}
}
