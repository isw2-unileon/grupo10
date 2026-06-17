package notes

import (
	"context"
	"database/sql"
	"errors"
)

// ErrNoteNotFound se devuelve cuando un apunte no existe en la base de datos.
var ErrNoteNotFound = errors.New("apunte no encontrado")

// Repository define las operaciones de base de datos para los apuntes.
type Repository interface {
	CreateNote(ctx context.Context, note *Note) error
	GetByAuthor(ctx context.Context, authorID string) ([]Note, error)
	GetByID(ctx context.Context, id string) (*Note, error)
	UpdateNote(ctx context.Context, note *Note) error
	UpdateStatus(ctx context.Context, noteID string, status NoteStatus) error
	UpdateNoteWithAI(ctx context.Context, noteID, feedback string, log *AIFeedbackLog) error
	DeleteNote(ctx context.Context, id string, authorID string) error
	GetPending(ctx context.Context) ([]Note, error)
	ApproveNoteWithFeedback(ctx context.Context, noteID, feedback string) error

	// Nuevas funciones para compartir
	GetUserEmail(ctx context.Context, userID string) (string, error)
	ShareNote(ctx context.Context, noteID string, email, groupID *string) error
	GetSharedWithMe(ctx context.Context, email string) ([]Note, error)
}

// PostgresRepository implementa la interfaz Repository usando PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository crea una nueva instancia de PostgresRepository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateNote inserta un nuevo apunte en la base de datos.
func (r *PostgresRepository) CreateNote(ctx context.Context, n *Note) error {
	query := `
		INSERT INTO notes (author_id, subject_id, title, content, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, n.AuthorID, n.SubjectID, n.Title, n.Content, n.Status).
		Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
}

// GetByAuthor obtiene todos los apuntes de un usuario específico.
//
//nolint:dupl
func (r *PostgresRepository) GetByAuthor(ctx context.Context, authorID string) ([]Note, error) {
	query := `SELECT id, author_id, subject_id, title, content, status, ai_feedback, teacher_feedback, created_at, updated_at 
	          FROM notes WHERE author_id = $1 ORDER BY updated_at DESC`
	rows, err := r.db.QueryContext(ctx, query, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.AuthorID, &n.SubjectID, &n.Title, &n.Content, &n.Status, &n.AiFeedback, &n.TeacherFeedback, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	return list, nil
}

// GetByID obtiene un apunte por su ID único.
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*Note, error) {
	var n Note
	query := `SELECT id, author_id, subject_id, title, content, status, ai_feedback FROM notes WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&n.ID, &n.AuthorID, &n.SubjectID, &n.Title, &n.Content, &n.Status, &n.AiFeedback)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoteNotFound
	}
	return &n, err
}

// UpdateNote modifica el título y el contenido de un apunte existente.
func (r *PostgresRepository) UpdateNote(ctx context.Context, n *Note) error {
	res, err := r.db.ExecContext(ctx, `UPDATE notes SET title = $1, content = $2, updated_at = NOW() WHERE id = $3 AND author_id = $4`, n.Title, n.Content, n.ID, n.AuthorID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNoteNotFound
	}
	return nil
}

// UpdateStatus cambia únicamente el estado de un apunte.
func (r *PostgresRepository) UpdateStatus(ctx context.Context, noteID string, status NoteStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notes SET status = $1, updated_at = NOW() WHERE id = $2`, status, noteID)
	return err
}

// UpdateNoteWithAI guarda el feedback de la IA y actualiza el estado en una transacción.
func (r *PostgresRepository) UpdateNoteWithAI(ctx context.Context, noteID, feedback string, log *AIFeedbackLog) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err = tx.ExecContext(ctx, `UPDATE notes SET status = $1, ai_feedback = $2, updated_at = NOW() WHERE id = $3`, StatusAiReviewed, feedback, noteID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO ai_feedback_logs (note_id, prompt_used, response) VALUES ($1, $2, $3)`, log.NoteID, log.PromptUsed, log.Response); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteNote elimina un apunte de forma permanente.
func (r *PostgresRepository) DeleteNote(ctx context.Context, id string, authorID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM notes WHERE id = $1 AND author_id = $2`, id, authorID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNoteNotFound
	}
	return nil
}

// GetPending devuelve todos los apuntes pendientes de revisión.
func (r *PostgresRepository) GetPending(ctx context.Context) ([]Note, error) {
	query := `SELECT id, author_id, subject_id, title, content, status, ai_feedback, teacher_feedback, created_at, updated_at 
	          FROM notes WHERE status = 'pending' ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.AuthorID, &n.SubjectID, &n.Title, &n.Content, &n.Status, &n.AiFeedback, &n.TeacherFeedback, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	return list, nil
}

// ApproveNoteWithFeedback guarda la corrección del docente (professor) y aprueba el apunte.
func (r *PostgresRepository) ApproveNoteWithFeedback(ctx context.Context, noteID, feedback string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notes SET status = 'approved', teacher_feedback = $1, updated_at = NOW() WHERE id = $2`, feedback, noteID)
	return err
}

// GetUserEmail obtiene el email de un usuario dado su ID.
func (r *PostgresRepository) GetUserEmail(ctx context.Context, userID string) (string, error) {
	var email string
	err := r.db.QueryRowContext(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)
	return email, err
}

// ShareNote inserta un registro para compartir el apunte.
func (r *PostgresRepository) ShareNote(ctx context.Context, noteID string, email, groupID *string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO note_shares (note_id, shared_with_email, shared_with_group)
		VALUES ($1, $2, $3)`, noteID, email, groupID)
	return err
}

// GetSharedWithMe obtiene los apuntes que otros han compartido con este usuario mediante email o grupo.
//
//nolint:dupl
func (r *PostgresRepository) GetSharedWithMe(ctx context.Context, email string) ([]Note, error) {
	query := `
		SELECT DISTINCT n.id, n.author_id, n.subject_id, n.title, n.content, n.status, n.ai_feedback, n.teacher_feedback, n.created_at, n.updated_at
		FROM notes n
		JOIN note_shares ns ON n.id = ns.note_id
		LEFT JOIN group_members gm ON ns.shared_with_group = gm.group_id
		WHERE ns.shared_with_email = $1 OR gm.email = $1
		ORDER BY n.updated_at DESC`
	rows, err := r.db.QueryContext(ctx, query, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.AuthorID, &n.SubjectID, &n.Title, &n.Content, &n.Status, &n.AiFeedback, &n.TeacherFeedback, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	return list, nil
}
