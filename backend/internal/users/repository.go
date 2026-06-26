package users

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

// Repository abstracts persistence so the service can be unit-tested without a
// real database.
type Repository interface {
	CreateUser(ctx context.Context, u *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	RoleIDByName(ctx context.Context, name string) (string, error)
}

// PostgresRepository is the production implementation backed by PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository builds a repository over an open *sql.DB.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// 1. Deja selectUserQuery solo para buscar por ID (manteniendo su lógica)
const selectUserQuery = `
    SELECT u.id, u.role_id, r.name, u.name, u.email, u.password_hash, u.created_at
    FROM users u
    -- Mismo cast a ::text que en selectUserByEmailQuery: en Render role_id y roles.id
    -- pueden tener tipos distintos (UUID vs VARCHAR) y el JOIN directo falla con un 500.
    JOIN roles r ON r.id::text = u.role_id::text`

// 2. Crea una consulta ESPECÍFICA para el login por Email, asegurando compatibilidad de tipos
const selectUserByEmailQuery = `
    SELECT u.id, u.role_id, r.name, u.name, u.email, u.password_hash, u.created_at
    FROM users u
    -- Usamos ::text en ambos lados para que, tengan el tipo que tengan en Render (UUID o Varchar), Postgres los compare como texto sin quejarse
    JOIN roles r ON r.id::text = u.role_id::text
    WHERE u.email = $1`

// RoleIDByName resolves a role name ("student"/"teacher") to its UUID.
func (r *PostgresRepository) RoleIDByName(ctx context.Context, name string) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM roles WHERE name = $1`, name).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrInvalidRole
	}
	return id, err
}

// CreateUser inserts a new user and fills in the generated ID and timestamps.
func (r *PostgresRepository) CreateUser(ctx context.Context, u *User) error {
	const q = `
		INSERT INTO users (role_id, name, email, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	err := r.db.QueryRowContext(ctx, q, u.RoleID, u.Name, u.Email, u.PasswordHash).
		Scan(&u.ID, &u.CreatedAt)
	if isUniqueViolation(err) {
		return ErrEmailTaken
	}
	return err
}

// GetByEmail returns the user with the given email, or ErrUserNotFound.
func (r *PostgresRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	// Usamos la nueva query limpia y directa
	return scanUser(r.db.QueryRowContext(ctx, selectUserByEmailQuery, email))
}

// GetByID returns the user with the given UUID, or ErrUserNotFound.
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*User, error) {
	return scanUser(r.db.QueryRowContext(ctx, selectUserQuery+` WHERE u.id = $1`, id))
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.RoleID, &u.Role, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// isUniqueViolation reports whether err is a PostgreSQL unique_violation (23505).
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}
