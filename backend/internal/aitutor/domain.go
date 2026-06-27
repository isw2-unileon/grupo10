package aitutor

import "context"

// FailedQuestionContext almacena el texto de la pregunta fallada y la respuesta para dar contexto a la IA.
type FailedQuestionContext struct {
	QuestionText string `json:"question_text"`
	WrongAnswer  string `json:"wrong_answer"`
	RightAnswer  string `json:"right_answer"`
}

// AIQuizOption estructura las opciones generadas por la IA.
type AIQuizOption struct {
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
}

// AIQuizQuestion representa una pregunta generada por Inteligencia Artificial.
type AIQuizQuestion struct {
	QuestionText string         `json:"question_text"`
	Options      []AIQuizOption `json:"options"`
	Explanation  string         `json:"explanation"`
}

// GeneratedAIQuiz es el paquete final que el alumno resolverá en el navegador.
type GeneratedAIQuiz struct {
	SectionID    string           `json:"section_id"`
	SectionTitle string           `json:"section_title"` // Se usará como título descriptivo general
	Questions    []AIQuizQuestion `json:"questions"`
}

// Repository operaciones de base de datos para el módulo de IA.
type Repository interface {
	GetSectionsTitles(ctx context.Context, sectionIDs []string) ([]string, error)
	GetFailedQuestionsContext(ctx context.Context, studentID string, sectionIDs []string) ([]FailedQuestionContext, error)
}
