package users

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 8
	maxNameLength     = 150
)

// Service holds the user-related business logic.
type Service struct {
	repo   Repository
	tokens TokenIssuer
}

// NewService wires the service with its dependencies.
func NewService(repo Repository, tokens TokenIssuer) *Service {
	return &Service{repo: repo, tokens: tokens}
}

// RegisterInput carries the data needed to create an account.
type RegisterInput struct {
	Name     string
	Email    string
	Password string
	Role     string
}

// Register validates the input, hashes the password, persists the user and
// returns it together with a freshly issued auth token.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*User, string, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = normalizeEmail(in.Email)
	in.Role = strings.TrimSpace(in.Role)

	if err := validateRegister(in); err != nil {
		return nil, "", err
	}

	roleID, err := s.repo.RoleIDByName(ctx, in.Role)
	if err != nil {
		return nil, "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	u := &User{
		RoleID:       roleID,
		Role:         in.Role,
		Name:         in.Name,
		Email:        in.Email,
		PasswordHash: string(hash),
	}
	if err := s.repo.CreateUser(ctx, u); err != nil {
		return nil, "", err
	}

	token, err := s.tokens.Issue(u)
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

// Authenticate verifies the credentials and returns the user with a new token.
// It returns ErrInvalidCredentials for both unknown emails and wrong passwords
// so callers cannot distinguish between the two.
func (s *Service) Authenticate(ctx context.Context, email, password string) (*User, string, error) {
	u, err := s.repo.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}

	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.tokens.Issue(u)
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

// ByID returns the user with the given ID, or ErrUserNotFound.
func (s *Service) ByID(ctx context.Context, id string) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func validateRegister(in RegisterInput) error {
	if in.Name == "" || len(in.Name) > maxNameLength {
		return fmt.Errorf("%w: name is required (max %d characters)", ErrValidation, maxNameLength)
	}
	if _, err := mail.ParseAddress(in.Email); err != nil {
		return fmt.Errorf("%w: a valid email is required", ErrValidation)
	}
	if len(in.Password) < minPasswordLength {
		return fmt.Errorf("%w: password must be at least %d characters", ErrValidation, minPasswordLength)
	}
	if in.Role != RoleStudent && in.Role != RoleTeacher {
		return ErrInvalidRole
	}
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
