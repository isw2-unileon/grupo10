package user

import (
	"context"
	"database/sql"
	"errors"
)

// PostgresRepository implementa la interfaz Repository usando una DB real
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository es el constructor
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateUser inserta un usuario real en la tabla 'users'
func (r *PostgresRepository) CreateUser(ctx context.Context, u *User) error {
	// Query que coincide con los campos de tu up.sql
	// Nota: Dejamos que Postgres genere el ID automáticamente si lo pusiste como SERIAL o UUID DEFAULT
	query := `
		INSERT INTO users (role_id, name, email, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id;
	`

	// Ejecutamos la query y guardamos el ID generado de vuelta en nuestro struct
	err := r.db.QueryRowContext(ctx, query, u.RoleID, u.Name, u.Email, u.Password).Scan(&u.ID)
	if err != nil {
		return err
	}

	return nil
}

// GetUserByEmail busca un usuario en la base de datos por su email
func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, role_id, name, email, password_hash
		FROM users
		WHERE email = $1;
	`

	var u User
	err := r.db.QueryRowContext(ctx, query, email).Scan(&u.ID, &u.RoleID, &u.Name, &u.Email, &u.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Si no encuentra nada, devolvemos nil sin error (tal como espera el servicio)
		}
		return nil, err // Si es otro error (ej: fallo de conexión), lo escupimos
	}

	return &u, nil
}
