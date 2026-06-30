package domain

import "errors"

var (
	ErrFormNotFound        = errors.New("form template not found")
	ErrSubmissionNotFound  = errors.New("form submission not found")
	ErrDocumentNotFound    = errors.New("document not found")
	ErrInvalidFormData     = errors.New("invalid form data")
	ErrFileTooLarge        = errors.New("file too large")
	ErrUnsupportedFileType = errors.New("unsupported file type")
)
