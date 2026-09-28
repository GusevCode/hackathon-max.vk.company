package domain

// NotificationKind identifies the business event that caused a notification.
// Keeping the kind in the message makes the broker stream observable and lets
// us add other notification channels without coupling them to task handlers.
type NotificationKind string

const (
	NotificationTaskAssigned  NotificationKind = "task.assigned"
	NotificationTaskSubmitted NotificationKind = "task.submitted"
	NotificationTaskReviewed  NotificationKind = "task.reviewed"
)

// Notification is a user-facing message delivered asynchronously through the
// broker. RecipientUserID is a MAX user ID, not a chat ID.
type Notification struct {
	ID              string
	Kind            NotificationKind
	RecipientUserID int64
	TaskID          string
	Text            string
	// Buttons are rendered in a separate message as the refreshed menu. They
	// are intentionally not attached to the notification text itself.
	Buttons []Button
	Photos  []Photo
}
