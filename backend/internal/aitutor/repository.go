package aitutor

import (
	"context"
	"database/sql"

	"github.com/lib/pq" // Utilizado para transformar arrays de strings nativos a Postgres
)

// PostgresRepository implementa Repository usando PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository inicializa el repositorio de IA.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// GetSectionsTitles extrae los nombres de todos los temas seleccionados.
func (r *PostgresRepository) GetSectionsTitles(ctx context.Context, sectionIDs []string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT title FROM group_sections WHERE id::text = ANY($1)`, pq.Array(sectionIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var titles []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err == nil {
			titles = append(titles, t)
		}
	}
	return titles, nil
}

// GetFailedQuestionsContext extrae todos los fallos del estudiante combinando el espectro de temas seleccionados.
func (r *PostgresRepository) GetFailedQuestionsContext(ctx context.Context, studentID string, sectionIDs []string) ([]FailedQuestionContext, error) {
	const q = `
		SELECT 
			qq.question_text, 
			qo_sel.option_text AS wrong_answer, 
			qo_corr.option_text AS right_answer
		FROM student_quiz_answers sqa
		JOIN quiz_questions qq ON qq.id = sqa.question_id
		JOIN group_resources res ON res.id = qq.resource_id AND res.section_id::text = ANY($2)
		JOIN quiz_options qo_sel ON qo_sel.id = sqa.option_id
		JOIN quiz_options qo_corr ON qo_corr.question_id = qq.id AND qo_corr.is_correct = true
		WHERE sqa.student_id = $1 AND qo_sel.is_correct = false`

	rows, err := r.db.QueryContext(ctx, q, studentID, pq.Array(sectionIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fallos []FailedQuestionContext
	for rows.Next() {
		var f FailedQuestionContext
		if err := rows.Scan(&f.QuestionText, &f.WrongAnswer, &f.RightAnswer); err == nil {
			fallos = append(fallos, f)
		}
	}
	return fallos, nil
}
