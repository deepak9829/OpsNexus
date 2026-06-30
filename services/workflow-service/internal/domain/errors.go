package domain

import "errors"

var (
	ErrCaseNotFound      = errors.New("case not found")
	ErrTaskNotFound      = errors.New("task not found")
	ErrWorkflowNotFound  = errors.New("workflow not found")
	ErrCommentNotFound   = errors.New("comment not found")
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrInvalidInput      = errors.New("invalid input")
	ErrPermissionDenied  = errors.New("permission denied")
)
