package store

import "errors"

// Sentinel errors returned by the comment store. The HTTP layer maps these to
// stable errorCodes; internal error text is never sent to clients verbatim.
var (
	// ErrCommentNotFound is returned when a comment (for the given target) does
	// not exist.
	ErrCommentNotFound       = errors.New("comment not found")
	ErrInvalidCommentTarget  = errors.New("invalid comment target")
	ErrInvalidCommentID      = errors.New("invalid comment id")
	ErrInvalidCommentContent = errors.New("invalid comment content")
	ErrInvalidCommentParent  = errors.New("invalid comment parent")
	ErrCommentParentNotFound = errors.New("comment parent not found")
	ErrCommentParentMismatch = errors.New("comment parent does not match target")
	ErrInvalidCommentUser    = errors.New("invalid comment user")
	ErrReportDetailTooLong   = errors.New("report detail is too long")
	ErrReportNotFound        = errors.New("report not found")
)
