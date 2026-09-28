package control

import "github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"

type SessionKind string

const (
	SessionInviteCode       SessionKind = "invite_code"
	SessionObjectName       SessionKind = "object_name"
	SessionObjectAddress    SessionKind = "object_address"
	SessionWorkTypeName     SessionKind = "work_type_name"
	SessionTaskTitle        SessionKind = "task_title"
	SessionTaskDescription  SessionKind = "task_description"
	SessionTaskDueDate      SessionKind = "task_due_date"
	SessionPhotoBefore      SessionKind = "photo_before"
	SessionPhotoAfter       SessionKind = "photo_after"
	SessionPhotoAfterRemark SessionKind = "photo_after_remark"
	SessionUnableReason     SessionKind = "unable_reason"
	SessionReworkComment    SessionKind = "rework_comment"
)

type Session struct {
	Kind       SessionKind
	Role       domain.Role
	TaskID     string
	Draft      TaskDraft
	ObjectName string
	Photos     []domain.Photo
}

type TaskDraft struct {
	Title       string
	Description string
	AssigneeID  string
	DueAt       string
	ObjectID    string
	WorkTypeID  string
}
