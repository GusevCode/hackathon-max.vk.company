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
	Text   string
}
