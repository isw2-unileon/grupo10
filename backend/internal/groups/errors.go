package groups

import "errors"

// Domain errors. Handlers map these to HTTP status codes.
var (
	ErrValidation     = errors.New("validation failed")
	ErrForbidden      = errors.New("not allowed to perform this action")
	ErrGroupNotFound  = errors.New("group not found")
	ErrMemberNotFound = errors.New("member not found")
)
