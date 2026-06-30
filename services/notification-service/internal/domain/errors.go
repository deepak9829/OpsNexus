package domain

import "errors"

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrAuditEventNotFound   = errors.New("audit event not found")
	ErrInvalidInput         = errors.New("invalid input")
)
