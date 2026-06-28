package groups

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
)

// PostgresRepository implementa Repository usando PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository devuelve un repositorio PostgreSQL.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const selectGroup = `SELECT id, name, owner_id, created_at FROM class_groups`

// AccountByID busca un usuario.
func (r *PostgresRepository) AccountByID(ctx context.Context, id string) (*Account, error) {
	// Cast a ::text en ambos lados del JOIN: en Render role_id (users) y roles.id
	// pueden tener tipos distintos (UUID vs VARCHAR) y el JOIN directo da 500.
	// Mismo arreglo que en users/repository.go.
	const q = `SELECT u.id, r.name, u.email FROM users u JOIN roles r ON r.id::text = u.role_id::text WHERE u.id = $1`
	var acc Account
	err := r.db.QueryRowContext(ctx, q, id).Scan(&acc.ID, &acc.Role, &acc.Email)
	if errors.Is(err, sql.ErrNoRows) || isInvalidUUID(err) {
		return nil, ErrForbidden
	}
	return &acc, err
}

// CreateGroup crea un grupo.
func (r *PostgresRepository) CreateGroup(ctx context.Context, g *Group) error {
	const q = `INSERT INTO class_groups (name, owner_id) VALUES ($1, $2) RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, q, g.Name, g.OwnerID).Scan(&g.ID, &g.CreatedAt)
}

// GroupByID busca grupo por ID.
func (r *PostgresRepository) GroupByID(ctx context.Context, id string) (*Group, error) {
	return scanGroup(r.db.QueryRowContext(ctx, selectGroup+` WHERE id = $1`, id))
}

// GroupsOwnedBy devuelve los de un docente.
func (r *PostgresRepository) GroupsOwnedBy(ctx context.Context, ownerID string) ([]Group, error) {
	return r.queryGroups(ctx, selectGroup+` WHERE owner_id = $1 ORDER BY created_at DESC`, ownerID)
}

// GroupsForEmail busca por correo.
func (r *PostgresRepository) GroupsForEmail(ctx context.Context, email string) ([]Group, error) {
	const q = `SELECT g.id, g.name, g.owner_id, g.created_at FROM class_groups g JOIN group_members m ON m.group_id = g.id WHERE m.email = $1 ORDER BY g.created_at DESC`
	return r.queryGroups(ctx, q, email)
}

// AddMembers matricula alumnos.
func (r *PostgresRepository) AddMembers(ctx context.Context, groupID string, emails []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	const q = `INSERT INTO group_members (group_id, email) VALUES ($1, $2) ON CONFLICT (group_id, email) DO NOTHING`
	for _, email := range emails {
		if _, err := tx.ExecContext(ctx, q, groupID, email); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListMembers saca los alumnos.
func (r *PostgresRepository) ListMembers(ctx context.Context, groupID string) ([]Member, error) {
	const q = `SELECT m.id, m.email, (u.id IS NOT NULL) AS registered, m.added_at FROM group_members m LEFT JOIN users u ON u.email = m.email WHERE m.group_id = $1 ORDER BY m.email`
	rows, err := r.db.QueryContext(ctx, q, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := []Member{}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Email, &m.Registered, &m.AddedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

// RemoveMember borra alumno.
func (r *PostgresRepository) RemoveMember(ctx context.Context, groupID, memberID string) error {
	const q = `DELETE FROM group_members WHERE id = $1 AND group_id = $2`
	res, err := r.db.ExecContext(ctx, q, memberID, groupID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if n == 0 {
		return ErrMemberNotFound
	}
	return err
}

// IsMember valida.
func (r *PostgresRepository) IsMember(ctx context.Context, groupID, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND email = $2)`, groupID, email).Scan(&exists)
	return exists, err
}

// CreateSection crea bloque.
func (r *PostgresRepository) CreateSection(ctx context.Context, sec *Section) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO group_sections (group_id, title, position) VALUES ($1, $2, $3) RETURNING id`, sec.GroupID, sec.Title, sec.Position).Scan(&sec.ID)
}

// UpdateSection edita nombre.
func (r *PostgresRepository) UpdateSection(ctx context.Context, sectionID, title string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE group_sections SET title = $1 WHERE id = $2`, title, sectionID)
	return err
}

// DeleteSection borra.
func (r *PostgresRepository) DeleteSection(ctx context.Context, sectionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM group_sections WHERE id = $1`, sectionID)
	return err
}

// GetSections saca temas.
func (r *PostgresRepository) GetSections(ctx context.Context, groupID string) ([]Section, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, group_id, title, position FROM group_sections WHERE group_id = $1 ORDER BY position ASC`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	secs := []Section{}
	for rows.Next() {
		var s Section
		_ = rows.Scan(&s.ID, &s.GroupID, &s.Title, &s.Position)
		secs = append(secs, s)
	}
	return secs, nil
}

// GetSectionGroup recupera grupo superior.
func (r *PostgresRepository) GetSectionGroup(ctx context.Context, sectionID string) (string, error) {
	var gID string
	err := r.db.QueryRowContext(ctx, `SELECT group_id FROM group_sections WHERE id = $1`, sectionID).Scan(&gID)
	return gID, err
}

// CreateResource crea un file.
func (r *PostgresRepository) CreateResource(ctx context.Context, res *Resource) error {
	const q = `INSERT INTO group_resources (section_id, type, title, content, file_path, due_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, q, res.SectionID, res.Type, res.Title, res.Content, res.FilePath, res.DueAt).Scan(&res.ID, &res.CreatedAt)
}

// UpdateResource actualiza.
func (r *PostgresRepository) UpdateResource(ctx context.Context, res *Resource) error {
	const q = `UPDATE group_resources SET title = $1, content = $2, due_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, q, res.Title, res.Content, res.DueAt, res.ID)
	return err
}

// DeleteResource purga.
func (r *PostgresRepository) DeleteResource(ctx context.Context, resourceID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM group_resources WHERE id = $1`, resourceID)
	return err
}

// GetResourceByID lo recupera.
func (r *PostgresRepository) GetResourceByID(ctx context.Context, resourceID string) (*Resource, error) {
	const q = `SELECT id, section_id, type, title, content, COALESCE(file_path,''), due_at FROM group_resources WHERE id = $1`
	var res Resource
	var due sql.NullTime
	err := r.db.QueryRowContext(ctx, q, resourceID).Scan(&res.ID, &res.SectionID, &res.Type, &res.Title, &res.Content, &res.FilePath, &due)
	if due.Valid {
		res.DueAt = &due.Time
	}
	return &res, err
}

// ListResourcesForSection recursos por tema.
func (r *PostgresRepository) ListResourcesForSection(ctx context.Context, sectionID string) ([]Resource, error) {
	const q = `SELECT id, section_id, type, title, content, COALESCE(file_path,''), due_at, created_at FROM group_resources WHERE section_id = $1 ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, q, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Resource{}
	for rows.Next() {
		var res Resource
		var due sql.NullTime
		_ = rows.Scan(&res.ID, &res.SectionID, &res.Type, &res.Title, &res.Content, &res.FilePath, &due, &res.CreatedAt)
		if due.Valid {
			res.DueAt = &due.Time
		}
		list = append(list, res)
	}
	return list, nil
}

// CreateQuizQuestion test.
func (r *PostgresRepository) CreateQuizQuestion(ctx context.Context, q *QuizQuestion) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO quiz_questions (resource_id, question_text, position) VALUES ($1, $2, $3) RETURNING id`, q.ResourceID, q.QuestionText, q.Position).Scan(&q.ID)
}

// CreateQuizOption opcion.
func (r *PostgresRepository) CreateQuizOption(ctx context.Context, opt *QuizOption) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO quiz_options (question_id, option_text, is_correct) VALUES ($1, $2, $3) RETURNING id`, opt.QuestionID, opt.OptionText, opt.IsCorrect).Scan(&opt.ID)
}

// GetQuizQuestions devuelve array.
//
//nolint:dupl
func (r *PostgresRepository) GetQuizQuestions(ctx context.Context, resourceID string) ([]QuizQuestion, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, resource_id, question_text, position FROM quiz_questions WHERE resource_id = $1 ORDER BY position`, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []QuizQuestion{}
	for rows.Next() {
		var q QuizQuestion
		if err := rows.Scan(&q.ID, &q.ResourceID, &q.QuestionText, &q.Position); err == nil {
			list = append(list, q)
		}
	}
	return list, nil
}

// GetQuizOptions devuelve array.
//
//nolint:dupl
func (r *PostgresRepository) GetQuizOptions(ctx context.Context, questionID string) ([]QuizOption, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, question_id, option_text, is_correct FROM quiz_options WHERE question_id = $1`, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []QuizOption{}
	for rows.Next() {
		var o QuizOption
		if err := rows.Scan(&o.ID, &o.QuestionID, &o.OptionText, &o.IsCorrect); err == nil {
			list = append(list, o)
		}
	}
	return list, nil
}

// SubmitAssignment envia notas.
func (r *PostgresRepository) SubmitAssignment(ctx context.Context, sub *Submission) error {
	const q = `INSERT INTO student_submissions (resource_id, student_id, text_content, file_path, grade, feedback) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (resource_id, student_id) DO UPDATE SET text_content = $3, file_path = $4, grade = $5, feedback = $6, submitted_at = NOW()`
	_, err := r.db.ExecContext(ctx, q, sub.ResourceID, sub.StudentID, sub.TextContent, sub.FilePath, sub.Grade, sub.Feedback)
	return err
}

// GradeSubmission califica.
func (r *PostgresRepository) GradeSubmission(ctx context.Context, resourceID, studentID string, grade float64, feedback string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE student_submissions SET grade = $1, feedback = $2 WHERE resource_id = $3 AND student_id = $4`, grade, feedback, resourceID, studentID)
	return err
}

// GetSubmissions listar todo.
func (r *PostgresRepository) GetSubmissions(ctx context.Context, resourceID string) ([]Submission, error) {
	const q = `SELECT s.id, s.resource_id, s.student_id, u.email, COALESCE(s.text_content,''), COALESCE(s.file_path,''), s.grade, COALESCE(s.feedback,''), s.submitted_at FROM student_submissions s JOIN users u ON u.id = s.student_id WHERE s.resource_id = $1`
	rows, err := r.db.QueryContext(ctx, q, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Submission{}
	for rows.Next() {
		var sub Submission
		var g sql.NullFloat64
		_ = rows.Scan(&sub.ID, &sub.ResourceID, &sub.StudentID, &sub.StudentEmail, &sub.TextContent, &sub.FilePath, &g, &sub.Feedback, &sub.SubmittedAt)
		if g.Valid {
			sub.Grade = &g.Float64
		}
		list = append(list, sub)
	}
	return list, nil
}

// HasSubmitted checkea si mandó algo.
func (r *PostgresRepository) HasSubmitted(ctx context.Context, resourceID, studentID string) (bool, time.Time, *float64, error) {
	var subAt time.Time
	var g sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `SELECT submitted_at, grade FROM student_submissions WHERE resource_id = $1 AND student_id = $2`, resourceID, studentID).Scan(&subAt, &g)
	if errors.Is(err, sql.ErrNoRows) {
		return false, time.Time{}, nil, nil
	}
	var gradePtr *float64
	if g.Valid {
		gradePtr = &g.Float64
	}
	return true, subAt, gradePtr, err
}

// SaveQuizAnswer guarda selección de alumno.
func (r *PostgresRepository) SaveQuizAnswer(ctx context.Context, resourceID, studentID, questionID, optionID string) error {
	const q = `INSERT INTO student_quiz_answers (resource_id, student_id, question_id, option_id) VALUES ($1, $2, $3, $4) ON CONFLICT (student_id, question_id) DO UPDATE SET option_id = $4`
	_, err := r.db.ExecContext(ctx, q, resourceID, studentID, questionID, optionID)
	return err
}

// GetStudentAnswers recupera diccionario test.
func (r *PostgresRepository) GetStudentAnswers(ctx context.Context, resourceID, studentID string) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT question_id, option_id FROM student_quiz_answers WHERE resource_id = $1 AND student_id = $2`, resourceID, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var qID, oID string
		if err := rows.Scan(&qID, &oID); err == nil {
			m[qID] = oID
		}
	}
	return m, nil
}

func (r *PostgresRepository) queryGroups(ctx context.Context, query string, args ...any) ([]Group, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	gs := []Group{}
	for rows.Next() {
		var g Group
		_ = rows.Scan(&g.ID, &g.Name, &g.OwnerID, &g.CreatedAt)
		gs = append(gs, g)
	}
	return gs, nil
}

func scanGroup(row *sql.Row) (*Group, error) {
	var g Group
	if err := row.Scan(&g.ID, &g.Name, &g.OwnerID, &g.CreatedAt); err != nil {
		return nil, ErrGroupNotFound
	}
	return &g, nil
}

func isInvalidUUID(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "22P02"
	}
	return false
}

// GetStudentAnalytics calcula las medias matemáticas del alumno cruzando entregas y cuestionarios con nota.
func (r *PostgresRepository) GetStudentAnalytics(ctx context.Context, studentID string) ([]SubjectStat, error) {
	const q = `
		SELECT 
			g.id::text AS group_id, g.name AS group_name,
			COALESCE(s.id::text, '') AS section_id, COALESCE(s.title, '') AS section_title,
			COALESCE(AVG(sub.grade), 0) AS avg_grade,
			COUNT(sub.grade) AS graded_count
		FROM class_groups g
		JOIN group_members gm ON gm.group_id = g.id
		JOIN users u ON LOWER(u.email) = LOWER(gm.email) AND (u.id::text = $1 OR gm.id::text = $1)
		LEFT JOIN group_sections s ON s.group_id = g.id
		LEFT JOIN group_resources res ON res.section_id = s.id AND res.type IN ('quiz', 'assignment')
		LEFT JOIN student_submissions sub ON sub.resource_id = res.id AND sub.student_id = u.id AND sub.grade IS NOT NULL
		GROUP BY g.id, g.name, s.id, s.title, s.position
		ORDER BY g.name, s.position ASC`

	rows, err := r.db.QueryContext(ctx, q, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groupsMap := make(map[string]*SubjectStat)
	var orderedGroupIDs []string

	for rows.Next() {
		var gID, gName, sID, sTitle string
		var avg float64
		var count int
		if err := rows.Scan(&gID, &gName, &sID, &sTitle, &avg, &count); err != nil {
			return nil, err
		}

		if _, exists := groupsMap[gID]; !exists {
			groupsMap[gID] = &SubjectStat{GroupID: gID, GroupName: gName, Sections: []SectionStat{}}
			orderedGroupIDs = append(orderedGroupIDs, gID)
		}

		if sID != "" {
			groupsMap[gID].Sections = append(groupsMap[gID].Sections, SectionStat{
				SectionID: sID, SectionTitle: sTitle, Average: avg, GradedCount: count,
			})
		}
	}

	out := []SubjectStat{}
	for _, id := range orderedGroupIDs {
		subStat := groupsMap[id]
		var sum float64
		var validSections int
		for _, sec := range subStat.Sections {
			if sec.GradedCount > 0 {
				sum += sec.Average
				validSections++
			}
		}
		if validSections > 0 {
			subStat.TotalAverage = sum / float64(validSections)
		}
		out = append(out, *subStat)
	}
	return out, nil
}
