package notes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Service contiene la lógica de negocio de los apuntes.
type Service struct {
	repo Repository
}

// NewService inicializa un nuevo servicio de apuntes.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateNote procesa la creación de un nuevo apunte.
func (s *Service) CreateNote(ctx context.Context, authorID string, title, content string) (*Note, error) {
	n := &Note{
		AuthorID:  authorID,
		SubjectID: nil, // Enviamos nulo ya que la BD ahora lo permite
		Title:     title,
		Content:   content,
		Status:    StatusDraft,
	}
	err := s.repo.CreateNote(ctx, n)
	return n, err
}

// GetNotesByAuthor recupera el listado de apuntes de un alumno.
func (s *Service) GetNotesByAuthor(ctx context.Context, authorID string) ([]Note, error) {
	return s.repo.GetByAuthor(ctx, authorID)
}

// UpdateNote procesa la modificación manual de un apunte.
func (s *Service) UpdateNote(ctx context.Context, id, authorID, title, content string) error {
	n := &Note{ID: id, AuthorID: authorID, Title: title, Content: content}
	return s.repo.UpdateNote(ctx, n)
}

// DeleteNote procesa el borrado seguro de un apunte.
func (s *Service) DeleteNote(ctx context.Context, id, authorID string) error {
	return s.repo.DeleteNote(ctx, id, authorID)
}

type openAIRequest struct {
	Model    string      `json:"model"`
	Messages []openAIMsg `json:"messages"`
}

type openAIMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMsg `json:"message"`
	} `json:"choices"`
}

// RequestAIReview envía el contenido a la IA para generar sugerencias de mejora.
func (s *Service) RequestAIReview(ctx context.Context, noteID, content string) error {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return errors.New("no hay clave de API configurada para la IA (Groq)")
	}

	prompt := "Eres un docente estricto pero amable. Corrige estos apuntes, señala errores ortográficos, " +
		"conceptuales y sugiere mejoras. Sé conciso y devuelve el texto bien formateado. Apuntes a corregir:\n\n" + content

	reqBody := openAIRequest{
		Model: "llama-3.3-70b-versatile",
		Messages: []openAIMsg{
			{Role: "system", Content: "Eres un tutor educativo de alto nivel. Responde siempre en español y sé muy conciso en tu respuesta."},
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error al empaquetar JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error al crear petición HTTP: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error conectando con la IA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("la IA devolvió un error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var aiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return errors.New("error leyendo respuesta de la IA")
	}

	if len(aiResp.Choices) == 0 {
		return errors.New("la IA no devolvió ningún texto")
	}

	feedback := aiResp.Choices[0].Message.Content

	log := &AIFeedbackLog{NoteID: noteID, PromptUsed: prompt, Response: feedback}
	return s.repo.UpdateNoteWithAI(ctx, noteID, feedback, log)
}

// SubmitForApproval marca el apunte como pendiente para el docente(professor).
func (s *Service) SubmitForApproval(ctx context.Context, noteID string) error {
	return s.repo.UpdateStatus(ctx, noteID, StatusPending)
}

// GetPendingForTeacher obtiene los apuntes que requieren revisión.
func (s *Service) GetPendingForTeacher(ctx context.Context) ([]Note, error) {
	return s.repo.GetPending(ctx)
}

// ApproveNoteWithFeedback procesa la corrección del docente(professor).
func (s *Service) ApproveNoteWithFeedback(ctx context.Context, noteID, feedback string) error {
	return s.repo.ApproveNoteWithFeedback(ctx, noteID, feedback)
}
