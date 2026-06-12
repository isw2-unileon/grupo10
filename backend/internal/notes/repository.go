package notes

import (
	"context"
	"database/sql"
	"errors"
)

var ErrNoteNotFound = errors.New("apunte no encontrado")

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
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateNote(ctx context.Context, n *Note) error {
	query := `
		INSERT INTO notes (author_id, subject_id, title, content, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, n.AuthorID, n.SubjectID, n.Title, n.Content, n.Status).
		Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
}

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

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*Note, error) {
	var n Note
	query := `SELECT id, author_id, subject_id, title, content, status, ai_feedback FROM notes WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&n.ID, &n.AuthorID, &n.SubjectID, &n.Title, &n.Content, &n.Status, &n.AiFeedback)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoteNotFound
	}
	return &n, err
}

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

func (r *PostgresRepository) UpdateStatus(ctx context.Context, noteID string, status NoteStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notes SET status = $1, updated_at = NOW() WHERE id = $2`, status, noteID)
	return err
}

func (r *PostgresRepository) UpdateNoteWithAI(ctx context.Context, noteID, feedback string, log *AIFeedbackLog) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(ctx, `UPDATE notes SET status = $1, ai_feedback = $2, updated_at = NOW() WHERE id = $3`, StatusAiReviewed, feedback, noteID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO ai_feedback_logs (note_id, prompt_used, response) VALUES ($1, $2, $3)`, log.NoteID, log.PromptUsed, log.Response); err != nil {
		return err
	}

	return tx.Commit()
}

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

func (r *PostgresRepository) ApproveNoteWithFeedback(ctx context.Context, noteID, feedback string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notes SET status = 'approved', teacher_feedback = $1, updated_at = NOW() WHERE id = $2`, feedback, noteID)
	return err
}
