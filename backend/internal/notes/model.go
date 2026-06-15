package notes

import "time"

// NoteStatus define el tipo para los estados de un apunte.
type NoteStatus string

const (
	// StatusDraft indica que el apunte es un borrador.
	StatusDraft NoteStatus = "draft"
	// StatusAiReviewed indica que la IA lo ha revisado.
	StatusAiReviewed NoteStatus = "ai_reviewed"
	// StatusPending indica que está esperando nota del docente(professor).
	StatusPending NoteStatus = "pending"
	// StatusApproved indica que el docente(professor) lo ha aprobado.
	StatusApproved NoteStatus = "approved"
)

// Note representa un apunte en la base de datos.
type Note struct {
	ID              string     `json:"id"`
	AuthorID        string     `json:"author_id"`
	SubjectID       *string    `json:"subject_id"` // Puntero para que pueda ser nulo
	Title           string     `json:"title"`
	Content         string     `json:"content"`
	Status          NoteStatus `json:"status"`
	AiFeedback      *string    `json:"ai_feedback,omitempty"`
	TeacherFeedback *string    `json:"teacher_feedback,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// AIFeedbackLog almacena el registro de peticiones hechas a la IA.
type AIFeedbackLog struct {
	ID         string    `json:"id"`
	NoteID     string    `json:"note_id"`
	PromptUsed string    `json:"prompt_used"`
	Response   string    `json:"response"`
	CreatedAt  time.Time `json:"created_at"`
}
