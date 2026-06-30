package domain

import "time"

type Document struct {
	ID             string    `json:"id" bson:"id"`
	TenantID       string    `json:"tenant_id" bson:"tenant_id"`
	Filename       string    `json:"filename" bson:"filename"`
	OriginalName   string    `json:"original_name" bson:"original_name"`
	MimeType       string    `json:"mime_type" bson:"mime_type"`
	SizeBytes      int64     `json:"size_bytes" bson:"size_bytes"`
	StorageKey     string    `json:"storage_key" bson:"storage_key"`
	UploadedBy     string    `json:"uploaded_by" bson:"uploaded_by"`
	CaseID         *string   `json:"case_id,omitempty" bson:"case_id,omitempty"`
	VersionCount   int       `json:"version_count" bson:"version_count"`
	CurrentVersion int       `json:"current_version" bson:"current_version"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" bson:"updated_at"`
}

type DocumentVersion struct {
	ID         string    `json:"id" bson:"id"`
	DocumentID string    `json:"document_id" bson:"document_id"`
	Version    int       `json:"version" bson:"version"`
	StorageKey string    `json:"storage_key" bson:"storage_key"`
	UploadedBy string    `json:"uploaded_by" bson:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at" bson:"created_at"`
}
