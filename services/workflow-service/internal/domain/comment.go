package domain

import "time"

type Comment struct {
	ID        string
	CaseID    string
	TenantID  string
	AuthorID  string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
