package users

import "time"

// Role names seeded by the migrations (see migrations/up.sql).
const (
	RoleStudent = "student"
	RoleTeacher = "teacher"
)

// User represents an account in the platform. PasswordHash is never exposed
// in JSON responses.
type User struct {
	ID           string    `json:"id"`
	RoleID       string    `json:"-"`
	Role         string    `json:"role"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
