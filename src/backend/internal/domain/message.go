package domain

// BotInfo is the small part of MAX bot metadata used by the application.
type BotInfo struct {
	ID       int64
	Name     string
	Username string
}

// Message is a normalized incoming chat message.
type Message struct {
	ChatID int64
	UserID int64
	Text   string
}

type Photo struct {
	URL       string
	Token     string
	ObjectKey string
}

type Event struct {
	Kind        EventKind
	ChatID      int64
	UserID      int64
	DisplayName string
	MessageID   string
	Text        string
	Photos      []Photo
	Payload     string
	CallbackID  string
}

type EventKind string

const (
	EventMessage  EventKind = "message"
	EventCallback EventKind = "callback"
	EventStarted  EventKind = "started"
)

type Button struct {
	Text    string
	Payload string
	Row     int
}

type OutgoingMessage struct {
	ChatID int64
	// UserID is used for a direct 1:1 message. ChatID remains the recipient
	// for messages sent into a conversation or when editing an existing one.
	UserID    int64
	MessageID string
	Text      string
	Buttons   []Button
	Photos    []Photo
}
