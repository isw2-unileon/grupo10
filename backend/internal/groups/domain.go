package groups

import (
	"context"
	"time"
)

// Group is a class group owned by a teacher (ADR-002).
type Group struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Member is a roster entry, keyed by email. Registered is computed (a user with
// that email exists), so the teacher can tell who has signed up and who is still
// pending.
type Member struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Registered bool      `json:"registered"`
	AddedAt    time.Time `json:"added_at"`
}

// Task is an item posted to a group. Description and DueAt are optional.
type Task struct {
	ID          string     `json:"id"`
	GroupID     string     `json:"group_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	DueAt       *time.Time `json:"due_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Account is the minimal identity the module needs about the authenticated
// user: the role gates teacher-only actions and the email resolves membership.
type Account struct {
	ID    string
	Role  string
	Email string
}

// Repository abstracts persistence so the service can be unit-tested without a
// real database.
type Repository interface {
	// AccountByID returns the role and email of the authenticated user.
	AccountByID(ctx context.Context, id string) (*Account, error)

	CreateGroup(ctx context.Context, g *Group) error
	GroupByID(ctx context.Context, id string) (*Group, error)
	GroupsOwnedBy(ctx context.Context, ownerID string) ([]Group, error)
	GroupsForEmail(ctx context.Context, email string) ([]Group, error)

	// AddMembers inserts the given emails into the group, ignoring duplicates.
	AddMembers(ctx context.Context, groupID string, emails []string) error
	ListMembers(ctx context.Context, groupID string) ([]Member, error)
	RemoveMember(ctx context.Context, groupID, memberID string) error
	IsMember(ctx context.Context, groupID, email string) (bool, error)

	CreateTask(ctx context.Context, t *Task) error
	ListTasks(ctx context.Context, groupID string) ([]Task, error)
}
