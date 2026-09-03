package store

import "errors"

// Sentinel errors returned by the prompt store. The message text of
// ErrPromptNotFound ("prompt not found") and ErrPromptForbidden ("forbidden") is
// part of the HTTP contract consumed by internal/api, so it must not change.
var (
	// ErrPromptNotFound is returned when a prompt (for the given owner) does not
	// exist or has been soft-deleted.
	ErrPromptNotFound = errors.New("prompt not found")
	// ErrPromptForbidden is returned when a caller tries to modify a prompt they
	// do not own.
	ErrPromptForbidden = errors.New("forbidden")
	// ErrInvalidCategory is returned when a prompt references an unknown category.
	ErrInvalidCategory = errors.New("invalid category")
	// ErrInvalidTag is returned when a supplied tag is blank or exceeds limits.
	ErrInvalidTag = errors.New("invalid tag")
	// ErrInvalidReportReason is returned when a report reason is not one of the
	// bounded, typed set (see report_reason.go).
	ErrInvalidReportReason = errors.New("invalid report reason")
	ErrAdminRequired       = errors.New("administrator role required")
	ErrInvalidModeration   = errors.New("invalid moderation action")
	ErrModerationNotFound  = errors.New("moderation target not found")
	ErrCannotModerateSelf  = errors.New("administrator cannot disable self")
)
