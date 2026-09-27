package domain

import "time"

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleManager  Role = "manager"
	RoleEmployee Role = "employee"
)

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

type Organization struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

type Invite struct {
	Code           string
	OrganizationID string
	Roles          []Role
	ManagerID      string
	ExpiresAt      time.Time
	UsedBy         int64
}

type Object struct {
	ID             string
	OrganizationID string
	Name           string
	Address        string
	Kind           string
}

type WorkType struct {
	ID             string
	OrganizationID string
	Name           string
}

type TaskStatus string

const (
	TaskCreated    TaskStatus = "created"
	TaskAssigned   TaskStatus = "assigned"
	TaskInProgress TaskStatus = "in_progress"
	TaskSubmitted  TaskStatus = "submitted"
	TaskAccepted   TaskStatus = "accepted"
	TaskRework     TaskStatus = "rework"
	TaskUnable     TaskStatus = "unable"
)

type Priority string

const (
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
)

type Task struct {
	ID             string
	OrganizationID string
	Title          string
	Description    string
	ObjectID       string
	WorkTypeID     string
	AssigneeID     string
	ManagerID      string
	DueAt          time.Time
	Priority       Priority
	Status         TaskStatus
	Comment        string
	BeforePhotos   []Photo
	AfterPhotos    []Photo
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Review struct {
	TaskID     string
	ReviewerID string
	Decision   string
	Comment    string
	CreatedAt  time.Time
}

type Evidence struct {
	ID        string
	TaskID    string
	Kind      string
	ObjectKey string
	CreatedAt time.Time
}
