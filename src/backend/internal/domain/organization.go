package domain

import "time"

type Organization struct {
	ID        string
	Name      string
	CreatedAt time.Time
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
