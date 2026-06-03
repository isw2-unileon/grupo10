package users

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeRepo is an in-memory Repository used to test the service without a DB.
type fakeRepo struct {
	usersByEmail map[string]*User
	roles        map[string]string
	nextID       int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		usersByEmail: make(map[string]*User),
		roles: map[string]string{
			RoleStudent: "role-student-uuid",
			RoleTeacher: "role-teacher-uuid",
		},
	}
}

func (f *fakeRepo) RoleIDByName(_ context.Context, name string) (string, error) {
	id, ok := f.roles[name]
	if !ok {
		return "", ErrInvalidRole
	}
	return id, nil
}

func (f *fakeRepo) CreateUser(_ context.Context, u *User) error {
	if _, exists := f.usersByEmail[u.Email]; exists {
		return ErrEmailTaken
	}
	f.nextID++
	u.ID = string(rune('0' + f.nextID))
	u.CreatedAt = time.Now()
	f.usersByEmail[u.Email] = u
	return nil
}

func (f *fakeRepo) GetByEmail(_ context.Context, email string) (*User, error) {
	u, ok := f.usersByEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (f *fakeRepo) GetByID(_ context.Context, id string) (*User, error) {
	for _, u := range f.usersByEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}

func newTestService() *Service {
	return NewService(newFakeRepo(), NewJWTIssuer("test-secret", time.Hour))
}

func validInput() RegisterInput {
	return RegisterInput{
		Name:     "Ada Lovelace",
		Email:    "ada@example.com",
		Password: "supersecret",
		Role:     RoleStudent,
	}
}

func TestRegister_Success(t *testing.T) {
	svc := newTestService()

	u, token, err := svc.Register(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID == "" {
		t.Error("expected a generated ID")
	}
	if u.PasswordHash == "" || u.PasswordHash == "supersecret" {
		t.Error("password must be stored hashed, not in plain text")
	}
	if token == "" {
		t.Error("expected a token to be issued")
	}
}

func TestRegister_NormalizesEmail(t *testing.T) {
	svc := newTestService()

	in := validInput()
	in.Email = "  ADA@Example.COM "
	u, _, err := svc.Register(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "ada@example.com" {
		t.Errorf("expected normalized email, got %q", u.Email)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := newTestService()

	if _, _, err := svc.Register(context.Background(), validInput()); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	_, _, err := svc.Register(context.Background(), validInput())
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}
}

func TestRegister_Validation(t *testing.T) {
	cases := map[string]func(*RegisterInput){
		"empty name":     func(in *RegisterInput) { in.Name = "" },
		"invalid email":  func(in *RegisterInput) { in.Email = "not-an-email" },
		"short password": func(in *RegisterInput) { in.Password = "short" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			svc := newTestService()
			in := validInput()
			mutate(&in)
			if _, _, err := svc.Register(context.Background(), in); !errors.Is(err, ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

func TestRegister_InvalidRole(t *testing.T) {
	svc := newTestService()

	in := validInput()
	in.Role = "admin"
	if _, _, err := svc.Register(context.Background(), in); !errors.Is(err, ErrInvalidRole) {
		t.Errorf("expected ErrInvalidRole, got %v", err)
	}
}

func TestAuthenticate_Success(t *testing.T) {
	svc := newTestService()
	if _, _, err := svc.Register(context.Background(), validInput()); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	u, token, err := svc.Authenticate(context.Background(), "ada@example.com", "supersecret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "ada@example.com" || token == "" {
		t.Error("expected authenticated user and token")
	}
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	svc := newTestService()
	if _, _, err := svc.Register(context.Background(), validInput()); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, _, err := svc.Authenticate(context.Background(), "ada@example.com", "wrongpass")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticate_UnknownEmail(t *testing.T) {
	svc := newTestService()

	_, _, err := svc.Authenticate(context.Background(), "nobody@example.com", "whatever")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}
