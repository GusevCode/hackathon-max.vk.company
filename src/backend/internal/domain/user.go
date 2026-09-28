package domain

import "time"

type UserStatus string

const (
	UserActive  UserStatus = "active"
	UserBlocked UserStatus = "blocked"
)

type User struct {
	ID             string
	OrganizationID string
	MaxUserID      int64
	DisplayName    string
	Roles          []Role
	Status         UserStatus
	ManagerID      string
	CreatedAt      time.Time
}

func (u User) HasRole(role Role) bool {
	for _, candidate := range u.Roles {
		if candidate == role {
			return true
		}
	}
	return false
}

func (u User) IsActive() bool { return u.Status == UserActive }

type Invite struct {
	Code           string
	OrganizationID string
	Roles          []Role
	ManagerID      string
	ExpiresAt      time.Time
	UsedBy         int64
}

func (i Invite) IsAvailable(now time.Time) bool {
	return i.UsedBy == 0 && now.Before(i.ExpiresAt)
}
