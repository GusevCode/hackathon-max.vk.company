package control

import "github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"

type SessionKind string

const (
	SessionInviteCode        SessionKind = "invite_code"
	SessionObjectName        SessionKind = "object_name"
	SessionObjectAddress     SessionKind = "object_address"
	SessionWorkTypeName      SessionKind = "work_type_name"
	SessionTaskTitle         SessionKind = "task_title"
	SessionTaskDescription   SessionKind = "task_description"
	SessionTaskDueDate       SessionKind = "task_due_date"
	SessionTaskEditTitle     SessionKind = "task_edit_title"
	SessionTaskEditDesc      SessionKind = "task_edit_description"
	SessionTaskEditDueDate   SessionKind = "task_edit_due_date"
	SessionPhotoBefore       SessionKind = "photo_before"
	SessionPhotoAfter        SessionKind = "photo_after"
	SessionPhotoAfterRemark  SessionKind = "photo_after_remark"
	SessionUnableReason      SessionKind = "unable_reason"
	SessionReworkComment     SessionKind = "rework_comment"
	SessionObjectEditName    SessionKind = "object_edit_name"
	SessionObjectEditAddress SessionKind = "object_edit_address"
	SessionWorkTypeEditName  SessionKind = "work_type_edit_name"
)

type Session struct {
	Kind        SessionKind
	Role        domain.Role
	TaskID      string
	ReferenceID string
	Draft       TaskDraft
	ObjectName  string
	Photos      []domain.Photo
}

type TaskDraft struct {
	Title       string
	Description string
	AssigneeID  string
	DueAt       string
	ObjectID    string
	WorkTypeID  string
}
