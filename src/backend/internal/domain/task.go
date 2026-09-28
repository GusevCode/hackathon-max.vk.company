package domain

import "time"

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
