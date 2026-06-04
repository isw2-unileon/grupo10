package user

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidRole        = errors.New("invalid role")
)

type AuthService struct {
	repo Repository
}

func NewAuthService(repo Repository) *AuthService {
	return &AuthService{
		repo: repo,
	}
}

// Register crea un nuevo usuario validando roles y encriptando la contraseña
func (s *AuthService) Register(ctx context.Context, name, email, password, role string) error {
	// 1. Validar que el rol sea correcto (Regla de negocio)
	if role != "student" && role != "teacher" {
		return ErrInvalidRole
	}

	// 2. Comprobar si el email ya existe
	existingUser, _ := s.repo.GetUserByEmail(ctx, email)
	if existingUser != nil {
		return ErrUserAlreadyExists
	}

	// 3. Encriptar la contraseña
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 4. Crear el usuario con el rol incluido
	newUser := &User{
		Name:     name,
		Email:    email,
		Password: string(hashedPassword),
		RoleID:   role,
	}

	return s.repo.CreateUser(ctx, newUser)
}

// Login verifica las credenciales y devuelve al usuario si son correctas
func (s *AuthService) Login(ctx context.Context, email, password string) (*User, error) {
	// 1. Buscamos si existe un usuario con ese email
	user, _ := s.repo.GetUserByEmail(ctx, email)
	if user == nil {
		// Por seguridad, siempre devolvemos "Credenciales inválidas"
		// No le decimos al atacante "El email no existe"
		return nil, ErrInvalidCredentials
	}

	// 2. Comparamos la contraseña en texto plano con el Hash guardado
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		// Si no coinciden, fallamos
		return nil, ErrInvalidCredentials
	}

	// 3. Si todo es correcto, devolvemos al usuario
	return user, nil
}
