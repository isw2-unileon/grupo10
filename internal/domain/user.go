package domain

import "errors"

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// User representa a un usuario en el sistema, coincidiendo con tu up.sql
type User struct {
	ID           string
	RoleID       string // Referencia a 'student' o 'teacher'
	Name         string
	Email        string
	PasswordHash string
}
