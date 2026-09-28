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
