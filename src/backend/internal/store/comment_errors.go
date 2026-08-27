package store

import "errors"

// Sentinel errors returned by the comment store. The HTTP layer maps these to
// stable errorCodes; internal error text is never sent to clients verbatim.
var (
	// ErrCommentNotFound is returned when a comment (for the given target) does
	// not exist.
	ErrCommentNotFound = errors.New("comment not found")
)
