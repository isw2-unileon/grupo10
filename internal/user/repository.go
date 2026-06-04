package user

import "context"

// Local User model to avoid depending on external module import path.
// Adjust fields as needed to match your domain.User definition.
type User struct {
	ID       string
	Name     string
	Email    string
	Password string
	RoleID   string // "student" o "teacher"
}

// Repository define las operaciones de base de datos para los usuarios
type Repository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
}
