package domain

import "time"

type CaseStatus string

const (
	CaseStatusNew        CaseStatus = "new"
	CaseStatusOpen       CaseStatus = "open"
	CaseStatusInProgress CaseStatus = "in_progress"
	CaseStatusPending    CaseStatus = "pending"
	CaseStatusResolved   CaseStatus = "resolved"
	CaseStatusClosed     CaseStatus = "closed"
)

type CasePriority string

const (
	PriorityLow      CasePriority = "low"
	PriorityMedium   CasePriority = "medium"
	PriorityHigh     CasePriority = "high"
	PriorityCritical CasePriority = "critical"
)

type Case struct {
	ID          string
	TenantID    string
	CaseNumber  string
	Title       string
	Description string
	Status      CaseStatus
	Priority    CasePriority
	AssigneeID  *string
	ReporterID  string
	WorkflowID  *string
	SLA         SLA
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ResolvedAt  *time.Time
	ClosedAt    *time.Time
}

type SLA struct {
	DueAt    *time.Time
	Breached bool
}

type CaseTransition struct {
	ID          string
	CaseID      string
	FromStatus  CaseStatus
	ToStatus    CaseStatus
	Reason      string
	PerformedBy string
	PerformedAt time.Time
}
