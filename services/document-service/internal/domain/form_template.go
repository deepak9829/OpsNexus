package domain

import "time"

type FieldType string

const (
	FieldTypeText        FieldType = "text"
	FieldTypeEmail       FieldType = "email"
	FieldTypeNumber      FieldType = "number"
	FieldTypeDate        FieldType = "date"
	FieldTypeSelect      FieldType = "select"
	FieldTypeMultiselect FieldType = "multiselect"
	FieldTypeFile        FieldType = "file"
	FieldTypeTextarea    FieldType = "textarea"
)

type FormStatus string

const (
	FormStatusDraft     FormStatus = "draft"
	FormStatusPublished FormStatus = "published"
	FormStatusArchived  FormStatus = "archived"
)

type FormField struct {
	Name        string          `json:"name" bson:"name"`
	Type        FieldType       `json:"type" bson:"type"`
	Label       string          `json:"label" bson:"label"`
	Required    bool            `json:"required" bson:"required"`
	Placeholder string          `json:"placeholder" bson:"placeholder"`
	Options     []string        `json:"options,omitempty" bson:"options,omitempty"`
	Validation  FieldValidation `json:"validation" bson:"validation"`
}

type FieldValidation struct {
	MinLength *int     `json:"min_length,omitempty" bson:"min_length,omitempty"`
	MaxLength *int     `json:"max_length,omitempty" bson:"max_length,omitempty"`
	Min       *float64 `json:"min,omitempty" bson:"min,omitempty"`
	Max       *float64 `json:"max,omitempty" bson:"max,omitempty"`
	Pattern   string   `json:"pattern,omitempty" bson:"pattern,omitempty"`
}

type FormTemplate struct {
	ID          string      `json:"id" bson:"id"`
	TenantID    string      `json:"tenant_id" bson:"tenant_id"`
	Name        string      `json:"name" bson:"name"`
	Description string      `json:"description" bson:"description"`
	Version     int         `json:"version" bson:"version"`
	Fields      []FormField `json:"fields" bson:"fields"`
	Status      FormStatus  `json:"status" bson:"status"`
	CreatedBy   string      `json:"created_by" bson:"created_by"`
	CreatedAt   time.Time   `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" bson:"updated_at"`
}
