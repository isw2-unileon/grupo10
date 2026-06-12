package notes

import "time"

type NoteStatus string

const (
	StatusDraft      NoteStatus = "draft"
	StatusAiReviewed NoteStatus = "ai_reviewed"
	StatusPending    NoteStatus = "pending"
	StatusApproved   NoteStatus = "approved"
)

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

type AIFeedbackLog struct {
	ID         string    `json:"id"`
	NoteID     string    `json:"note_id"`
	PromptUsed string    `json:"prompt_used"`
	Response   string    `json:"response"`
	CreatedAt  time.Time `json:"created_at"`
}
