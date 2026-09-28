package domain

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleManager  Role = "manager"
	RoleEmployee Role = "employee"
)

func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleOperator, RoleManager, RoleEmployee:
		return true
	default:
		return false
	}
}

func (r Role) Label() string {
	switch r {
	case RoleAdmin:
		return "Администратор"
	case RoleOperator:
		return "Управляющий"
	case RoleManager:
		return "Руководитель"
	case RoleEmployee:
		return "Сотрудник"
	default:
		return "Неизвестная роль"
	}
}
