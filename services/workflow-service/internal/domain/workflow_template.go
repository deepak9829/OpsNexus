package domain

import "time"

type WorkflowTemplate struct {
	ID              string
	TenantID        string
	Name            string
	Description     string
	States          []string
	Transitions     []Transition
	DefaultPriority CasePriority
	SLAHours        int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Transition struct {
	From  CaseStatus
	To    CaseStatus
	Label string
}
