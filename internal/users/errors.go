package users

import "errors"

// Domain errors. Handlers map these to HTTP status codes.
var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidRole        = errors.New("role must be 'student' or 'teacher'")
	ErrUserNotFound       = errors.New("user not found")
	ErrValidation         = errors.New("validation failed")
)
