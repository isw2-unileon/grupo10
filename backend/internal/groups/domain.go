package groups

import (
	"context"
	"time"
)

// Group representa la asignatura.
type Group struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Member representa a un alumno matriculado.
type Member struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Registered bool      `json:"registered"`
	AddedAt    time.Time `json:"added_at"`
}

// Section representa un bloque de contenido.
type Section struct {
	ID        string     `json:"id"`
	GroupID   string     `json:"group_id"`
	Title     string     `json:"title"`
	Position  int        `json:"position"`
	Resources []Resource `json:"resources"`
}

// QuizQuestion estructura para las preguntas del cuestionario.
type QuizQuestion struct {
	ID           string       `json:"id"`
	ResourceID   string       `json:"resource_id"`
	QuestionText string       `json:"question_text"`
	Position     int          `json:"position"`
	Options      []QuizOption `json:"options"`
}

// QuizOption opciones A, B, C, D de cada pregunta.
type QuizOption struct {
	ID         string `json:"id"`
	QuestionID string `json:"question_id"`
	OptionText string `json:"option_text"`
	IsCorrect  bool   `json:"is_correct"`
	Selected   bool   `json:"selected"`
}

// Submission representa la entrega física de un alumno para que el docente (professor) la califique.
type Submission struct {
	ID           string    `json:"id"`
	ResourceID   string    `json:"resource_id"`
	StudentID    string    `json:"student_id"`
	StudentEmail string    `json:"student_email,omitempty"`
	TextContent  string    `json:"text_content"`
	FilePath     string    `json:"file_path"`
	Grade        *float64  `json:"grade,omitempty"`
	Feedback     string    `json:"feedback"`
	SubmittedAt  time.Time `json:"submitted_at"`
}

// Resource representa un archivo, tarea o cuestionario.
type Resource struct {
	ID        string     `json:"id"`
	SectionID string     `json:"section_id"`
	Type      string     `json:"type"` // 'file', 'assignment', 'quiz'
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	FilePath  string     `json:"file_path,omitempty"`
	DueAt     *time.Time `json:"due_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`

	// Estructura opcional si el recurso es un cuestionario
	Questions []QuizQuestion `json:"questions,omitempty"`

	// Campos de estado calculados para el estudiante logueado
	IsLate       bool       `json:"is_late"`
	HasSubmitted bool       `json:"has_submitted"`
	SubmittedAt  *time.Time `json:"submitted_at,omitempty"`
	CurrentGrade *float64   `json:"current_grade,omitempty"`
}

// Account identidad para el middleware.
type Account struct {
	ID    string
	Role  string
	Email string
}

// SectionStat detalla la nota media de un alumno en un tema específico.
type SectionStat struct {
	SectionID    string  `json:"section_id"`
	SectionTitle string  `json:"section_title"`
	Average      float64 `json:"average"`
	GradedCount  int     `json:"graded_count"`
}

// SubjectStat resume el rendimiento general en una asignatura completa.
type SubjectStat struct {
	GroupID      string        `json:"group_id"`
	GroupName    string        `json:"group_name"`
	TotalAverage float64       `json:"total_average"`
	Sections     []SectionStat `json:"sections"`
}

// StudentProfile agrupa los datos personales y académicos del alumno.
type StudentProfile struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Email     string        `json:"email"`
	Role      string        `json:"role"`
	Analytics []SubjectStat `json:"analytics"`
}

// Repository operaciones de persistencia del Moodle Avanzado.
type Repository interface {
	AccountByID(ctx context.Context, id string) (*Account, error)
	CreateGroup(ctx context.Context, g *Group) error
	GroupByID(ctx context.Context, id string) (*Group, error)
	GroupsOwnedBy(ctx context.Context, ownerID string) ([]Group, error)
	GroupsForEmail(ctx context.Context, email string) ([]Group, error)
	AddMembers(ctx context.Context, groupID string, emails []string) error
	ListMembers(ctx context.Context, groupID string) ([]Member, error)
	RemoveMember(ctx context.Context, groupID, memberID string) error
	IsMember(ctx context.Context, groupID, email string) (bool, error)

	// CRUD Secciones
	CreateSection(ctx context.Context, sec *Section) error
	UpdateSection(ctx context.Context, sectionID, title string) error
	DeleteSection(ctx context.Context, sectionID string) error
	GetSections(ctx context.Context, groupID string) ([]Section, error)
	GetSectionGroup(ctx context.Context, sectionID string) (string, error)

	// CRUD Recursos y Cuestionarios
	CreateResource(ctx context.Context, res *Resource) error
	UpdateResource(ctx context.Context, res *Resource) error
	DeleteResource(ctx context.Context, resourceID string) error
	GetResourceByID(ctx context.Context, resourceID string) (*Resource, error)
	ListResourcesForSection(ctx context.Context, sectionID string) ([]Resource, error)

	// Gestión de Cuestionarios (Preguntas e hilos de opciones)
	CreateQuizQuestion(ctx context.Context, q *QuizQuestion) error
	CreateQuizOption(ctx context.Context, opt *QuizOption) error
	GetQuizQuestions(ctx context.Context, resourceID string) ([]QuizQuestion, error)
	GetQuizOptions(ctx context.Context, questionID string) ([]QuizOption, error)
	// Persistencia de respuestas de cuestionarios
	SaveQuizAnswer(ctx context.Context, resourceID, studentID, questionID, optionID string) error
	GetStudentAnswers(ctx context.Context, resourceID, studentID string) (map[string]string, error)

	// Entregas y Notas de Alumnos
	SubmitAssignment(ctx context.Context, sub *Submission) error
	GradeSubmission(ctx context.Context, resourceID, studentID string, grade float64, feedback string) error
	GetSubmissions(ctx context.Context, resourceID string) ([]Submission, error)
	HasSubmitted(ctx context.Context, resourceID, studentID string) (bool, time.Time, *float64, error)

	GetStudentAnalytics(ctx context.Context, studentID string) ([]SubjectStat, error)
}
