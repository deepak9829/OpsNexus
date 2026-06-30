package domain

import "time"

type SubmissionStatus string

const (
	SubmissionStatusSubmitted  SubmissionStatus = "submitted"
	SubmissionStatusProcessing SubmissionStatus = "processing"
	SubmissionStatusApproved   SubmissionStatus = "approved"
	SubmissionStatusRejected   SubmissionStatus = "rejected"
)

type FormSubmission struct {
	ID          string           `json:"id" bson:"id"`
	FormID      string           `json:"form_id" bson:"form_id"`
	TenantID    string           `json:"tenant_id" bson:"tenant_id"`
	SubmittedBy string           `json:"submitted_by" bson:"submitted_by"`
	Data        map[string]any   `json:"data" bson:"data"`
	Status      SubmissionStatus `json:"status" bson:"status"`
	CreatedAt   time.Time        `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at" bson:"updated_at"`
}
