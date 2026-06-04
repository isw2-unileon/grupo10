package user

import (
	"context"
	"testing"
)

// 1. Creamos un Mock del Repositorio para no usar la base de datos real en los tests
type mockUserRepository struct {
	users map[string]*User
}

func (m *mockUserRepository) CreateUser(ctx context.Context, u *User) error {
	m.users[u.Email] = u
	return nil
}

func (m *mockUserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u, exists := m.users[email]
	if !exists {
		return nil, nil
	}
	return u, nil
}

func TestRegisterUser(t *testing.T) {
	repo := &mockUserRepository{
		users: make(map[string]*User),
	}
	service := NewAuthService(repo)

	t.Run("Registro exitoso", func(t *testing.T) {
		// Le pasamos "student" al final
		err := service.Register(context.Background(), "Juan", "juan@unileon.es", "password123", "student")
		if err != nil {
			t.Errorf("Se esperaba éxito, pero dio error: %v", err)
		}
	})

	t.Run("Fallo por email duplicado", func(t *testing.T) {
		_ = service.Register(context.Background(), "Juan", "juan2@unileon.es", "password123", "student")
		err := service.Register(context.Background(), "Copia Juan", "juan2@unileon.es", "password123", "student")

		if err != ErrUserAlreadyExists {
			t.Errorf("Se esperaba ErrUserAlreadyExists, pero se obtuvo: %v", err)
		}
	})

	t.Run("Fallo por rol invalido", func(t *testing.T) {
		// Intentamos registrar un "admin"
		err := service.Register(context.Background(), "Hacker", "hacker@unileon.es", "password123", "admin")

		if err != ErrInvalidRole {
			t.Errorf("Se esperaba ErrInvalidRole, pero se obtuvo: %v", err)
		}
	})
}

func TestLoginUser(t *testing.T) {
	repo := &mockUserRepository{
		users: make(map[string]*User),
	}
	service := NewAuthService(repo)

	// Registramos a un usuario válido previamente para poder probar el login
	_ = service.Register(context.Background(), "Ana", "ana@unileon.es", "secreta123", "student")

	t.Run("Login exitoso", func(t *testing.T) {
		// Intentamos hacer login con las credenciales correctas
		user, err := service.Login(context.Background(), "ana@unileon.es", "secreta123")
		if err != nil {
			t.Errorf("No se esperaba error, pero se obtuvo: %v", err)
		}
		if user == nil || user.Name != "Ana" {
			t.Errorf("Se esperaba recuperar al usuario Ana, pero falló")
		}
	})

	t.Run("Fallo por contraseña incorrecta", func(t *testing.T) {
		// Intentamos hacer login con una contraseña falsa
		_, err := service.Login(context.Background(), "ana@unileon.es", "contraseña_falsa")

		if err != ErrInvalidCredentials {
			t.Errorf("Se esperaba ErrInvalidCredentials, pero se obtuvo: %v", err)
		}
	})

	t.Run("Fallo por usuario no registrado", func(t *testing.T) {
		// Intentamos hacer login con un correo que no existe
		_, err := service.Login(context.Background(), "fantasma@unileon.es", "secreta123")

		if err != ErrInvalidCredentials {
			t.Errorf("Se esperaba ErrInvalidCredentials, pero se obtuvo: %v", err)
		}
	})
}
