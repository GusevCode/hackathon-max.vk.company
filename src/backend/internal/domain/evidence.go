package domain

import "time"

type Evidence struct {
	ID           string
	TaskID       string
	Kind         string
	ObjectKey    string
	SubmissionID string
	CreatedAt    time.Time
}
