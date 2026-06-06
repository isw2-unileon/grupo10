package groups

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

const (
	maxNameLength  = 200
	maxTitleLength = 300
	roleTeacher    = "teacher"
)

// Service holds the group-related business logic.
type Service struct {
	repo Repository
}

// NewService wires the service with its dependencies.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateGroup creates a group owned by the authenticated teacher.
func (s *Service) CreateGroup(ctx context.Context, userID, name string) (*Group, error) {
	acc, err := s.requireTeacher(ctx, userID)
	if err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxNameLength {
		return nil, fmt.Errorf("%w: name is required (max %d characters)", ErrValidation, maxNameLength)
	}

	g := &Group{Name: name, OwnerID: acc.ID}
	if err := s.repo.CreateGroup(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

// GroupsOwned lists the groups owned by the authenticated teacher.
func (s *Service) GroupsOwned(ctx context.Context, userID string) ([]Group, error) {
	acc, err := s.requireTeacher(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.GroupsOwnedBy(ctx, acc.ID)
}

// MyGroups lists the groups the authenticated user belongs to as a student,
// resolved by email. An empty result drives the "waiting to be added" state.
func (s *Service) MyGroups(ctx context.Context, userID string) ([]Group, error) {
	acc, err := s.repo.AccountByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.GroupsForEmail(ctx, acc.Email)
}

// GroupDetail returns the group together with its roster and tasks. Only the
// owner may see the roster.
func (s *Service) GroupDetail(ctx context.Context, userID, groupID string) (*Group, []Member, []Task, error) {
	g, err := s.ownedGroup(ctx, userID, groupID)
	if err != nil {
		return nil, nil, nil, err
	}

	members, err := s.repo.ListMembers(ctx, g.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	tasks, err := s.repo.ListTasks(ctx, g.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	return g, members, tasks, nil
}

// AddMembers adds the given emails to the group roster (owner only). Emails are
// normalised and de-duplicated; duplicates already in the roster are ignored.
// It returns the updated roster.
func (s *Service) AddMembers(ctx context.Context, userID, groupID string, emails []string) ([]Member, error) {
	g, err := s.ownedGroup(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}

	clean, err := sanitizeEmails(emails)
	if err != nil {
		return nil, err
	}
	if err := s.repo.AddMembers(ctx, g.ID, clean); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, g.ID)
}

// RemoveMember removes a roster entry from the group (owner only).
func (s *Service) RemoveMember(ctx context.Context, userID, groupID, memberID string) error {
	g, err := s.ownedGroup(ctx, userID, groupID)
	if err != nil {
		return err
	}
	return s.repo.RemoveMember(ctx, g.ID, memberID)
}

// CreateTask posts a task to the group (owner only).
func (s *Service) CreateTask(ctx context.Context, userID, groupID string, in TaskInput) (*Task, error) {
	g, err := s.ownedGroup(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(in.Title)
	if title == "" || len(title) > maxTitleLength {
		return nil, fmt.Errorf("%w: title is required (max %d characters)", ErrValidation, maxTitleLength)
	}

	t := &Task{GroupID: g.ID, Title: title, Description: in.Description, DueAt: in.DueAt}
	if err := s.repo.CreateTask(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// ListTasks returns a group's tasks. The caller must be the owner or a member.
func (s *Service) ListTasks(ctx context.Context, userID, groupID string) ([]Task, error) {
	acc, err := s.repo.AccountByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	g, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	if g.OwnerID != acc.ID {
		member, err := s.repo.IsMember(ctx, g.ID, acc.Email)
		if err != nil {
			return nil, err
		}
		if !member {
			return nil, ErrForbidden
		}
	}
	return s.repo.ListTasks(ctx, g.ID)
}

// TaskInput carries the data needed to create a task.
type TaskInput struct {
	Title       string
	Description *string
	DueAt       *time.Time
}

// requireTeacher loads the account and rejects non-teachers.
func (s *Service) requireTeacher(ctx context.Context, userID string) (*Account, error) {
	acc, err := s.repo.AccountByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if acc.Role != roleTeacher {
		return nil, ErrForbidden
	}
	return acc, nil
}

// ownedGroup loads the group and asserts the caller is its owner (which implies
// being a teacher). Returns ErrForbidden for any other user.
func (s *Service) ownedGroup(ctx context.Context, userID, groupID string) (*Group, error) {
	acc, err := s.repo.AccountByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	g, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if g.OwnerID != acc.ID {
		return nil, ErrForbidden
	}
	return g, nil
}

// sanitizeEmails normalises, validates and de-duplicates a list of emails,
// preserving order. It rejects the batch if no valid email remains.
func sanitizeEmails(emails []string) ([]string, error) {
	seen := make(map[string]struct{}, len(emails))
	clean := make([]string, 0, len(emails))
	for _, e := range emails {
		e = normalizeEmail(e)
		if e == "" {
			continue
		}
		if _, err := mail.ParseAddress(e); err != nil {
			return nil, fmt.Errorf("%w: %q is not a valid email", ErrValidation, e)
		}
		if _, dup := seen[e]; dup {
			continue
		}
		seen[e] = struct{}{}
		clean = append(clean, e)
	}
	if len(clean) == 0 {
		return nil, fmt.Errorf("%w: at least one valid email is required", ErrValidation)
	}
	return clean, nil
}

// normalizeEmail mirrors the users module so roster emails match user emails.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
