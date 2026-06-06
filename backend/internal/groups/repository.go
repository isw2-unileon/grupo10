package groups

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

// PostgresRepository is the production implementation backed by PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository builds a repository over an open *sql.DB.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const selectGroup = `SELECT id, name, owner_id, created_at FROM class_groups`

// AccountByID returns the role and email of the authenticated user.
func (r *PostgresRepository) AccountByID(ctx context.Context, id string) (*Account, error) {
	const q = `
		SELECT u.id, r.name, u.email
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1`
	var acc Account
	err := r.db.QueryRowContext(ctx, q, id).Scan(&acc.ID, &acc.Role, &acc.Email)
	if errors.Is(err, sql.ErrNoRows) || isInvalidUUID(err) {
		// The token references a user that no longer exists: deny the action.
		return nil, ErrForbidden
	}
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

// CreateGroup inserts a new group and fills in the generated ID and timestamp.
func (r *PostgresRepository) CreateGroup(ctx context.Context, g *Group) error {
	const q = `
		INSERT INTO class_groups (name, owner_id)
		VALUES ($1, $2)
		RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, q, g.Name, g.OwnerID).Scan(&g.ID, &g.CreatedAt)
}

// GroupByID returns the group with the given ID, or ErrGroupNotFound.
func (r *PostgresRepository) GroupByID(ctx context.Context, id string) (*Group, error) {
	return scanGroup(r.db.QueryRowContext(ctx, selectGroup+` WHERE id = $1`, id))
}

// GroupsOwnedBy lists the groups owned by a teacher, newest first.
func (r *PostgresRepository) GroupsOwnedBy(ctx context.Context, ownerID string) ([]Group, error) {
	return r.queryGroups(ctx, selectGroup+` WHERE owner_id = $1 ORDER BY created_at DESC`, ownerID)
}

// GroupsForEmail lists the groups whose roster contains the given email.
func (r *PostgresRepository) GroupsForEmail(ctx context.Context, email string) ([]Group, error) {
	const q = `
		SELECT g.id, g.name, g.owner_id, g.created_at
		FROM class_groups g
		JOIN group_members m ON m.group_id = g.id
		WHERE m.email = $1
		ORDER BY g.created_at DESC`
	return r.queryGroups(ctx, q, email)
}

// AddMembers inserts the given emails into the roster, ignoring duplicates.
func (r *PostgresRepository) AddMembers(ctx context.Context, groupID string, emails []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	const q = `
		INSERT INTO group_members (group_id, email)
		VALUES ($1, $2)
		ON CONFLICT (group_id, email) DO NOTHING`
	for _, email := range emails {
		if _, err := tx.ExecContext(ctx, q, groupID, email); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListMembers returns the roster with each email flagged registered or pending.
func (r *PostgresRepository) ListMembers(ctx context.Context, groupID string) ([]Member, error) {
	const q = `
		SELECT m.id, m.email, (u.id IS NOT NULL) AS registered, m.added_at
		FROM group_members m
		LEFT JOIN users u ON u.email = m.email
		WHERE m.group_id = $1
		ORDER BY m.email`
	rows, err := r.db.QueryContext(ctx, q, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Email, &m.Registered, &m.AddedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// RemoveMember deletes a roster entry, or returns ErrMemberNotFound.
func (r *PostgresRepository) RemoveMember(ctx context.Context, groupID, memberID string) error {
	const q = `DELETE FROM group_members WHERE id = $1 AND group_id = $2`
	res, err := r.db.ExecContext(ctx, q, memberID, groupID)
	if isInvalidUUID(err) {
		return ErrMemberNotFound
	}
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrMemberNotFound
	}
	return nil
}

// IsMember reports whether an email is on the group's roster.
func (r *PostgresRepository) IsMember(ctx context.Context, groupID, email string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND email = $2)`
	var exists bool
	err := r.db.QueryRowContext(ctx, q, groupID, email).Scan(&exists)
	return exists, err
}

// CreateTask inserts a new task and fills in the generated ID and timestamp.
func (r *PostgresRepository) CreateTask(ctx context.Context, t *Task) error {
	const q = `
		INSERT INTO group_tasks (group_id, title, description, due_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, q, t.GroupID, t.Title, t.Description, t.DueAt).
		Scan(&t.ID, &t.CreatedAt)
}

// ListTasks returns a group's tasks, ordered by due date (then creation).
func (r *PostgresRepository) ListTasks(ctx context.Context, groupID string) ([]Task, error) {
	const q = `
		SELECT id, group_id, title, description, due_at, created_at
		FROM group_tasks
		WHERE group_id = $1
		ORDER BY COALESCE(due_at, created_at) ASC`
	rows, err := r.db.QueryContext(ctx, q, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var (
			t    Task
			desc sql.NullString
			due  sql.NullTime
		)
		if err := rows.Scan(&t.ID, &t.GroupID, &t.Title, &desc, &due, &t.CreatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			t.Description = &desc.String
		}
		if due.Valid {
			t.DueAt = &due.Time
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *PostgresRepository) queryGroups(ctx context.Context, query string, args ...any) ([]Group, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gs []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.OwnerID, &g.CreatedAt); err != nil {
			return nil, err
		}
		gs = append(gs, g)
	}
	return gs, rows.Err()
}

func scanGroup(row *sql.Row) (*Group, error) {
	var g Group
	err := row.Scan(&g.ID, &g.Name, &g.OwnerID, &g.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		// A malformed UUID in the path reaches Postgres as a type error; treat
		// it as "not found" rather than a 500.
		if isInvalidUUID(err) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	return &g, nil
}

// isInvalidUUID reports whether err is a PostgreSQL invalid_text_representation
// (22P02), raised when a path parameter is not a valid UUID.
func isInvalidUUID(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "22P02"
	}
	return false
}
