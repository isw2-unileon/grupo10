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

// defaultGroqURL es el endpoint de la API de Groq usado en producción.
const defaultGroqURL = "https://api.groq.com/openai/v1/chat/completions"

// Service contiene la lógica de negocio de los apuntes.
type Service struct {
	repo  Repository
	aiURL string
}

// NewService inicializa un nuevo servicio de apuntes.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, aiURL: defaultGroqURL}
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

	systemPrompt := "You are an academic tutor integrated into Learning Platform, a university " +
		"application where students write and upload their notes to review them. Your role is to " +
		"review a university student's notes and help them improve before submitting or sharing them.\n\n" +
		"When reviewing:\n" +
		"- Fix spelling, grammar and wording mistakes.\n" +
		"- Point out conceptual errors or inaccuracies and explain them briefly.\n" +
		"- Suggest improvements or important concepts that are missing.\n" +
		"- Acknowledge what is well explained, with a demanding but warm and motivating tone.\n\n" +
		"Response rules: ALWAYS write your answer in Spanish, regardless of the language of these " +
		"instructions. Be concise and direct, and organize the response with clear sections or bullet " +
		"points. If the text is empty or does not look like academic notes, say so politely instead " +
		"of inventing content."

	prompt := "Review the following notes from a university student:\n\n" + content

	reqBody := openAIRequest{
		Model: "llama-3.3-70b-versatile",
		Messages: []openAIMsg{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error al empaquetar JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.aiURL, bytes.NewBuffer(jsonData))
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

// ShareNote valida la propiedad del apunte y tras ello, va a compartirlo.
func (s *Service) ShareNote(ctx context.Context, noteID, authorID string, email, groupID *string) error {
	// Verificamos que el apunte sea del usuario que lo intenta compartir
	note, err := s.repo.GetByID(ctx, noteID)
	if err != nil {
		return err
	}
	if note.AuthorID != authorID {
		return errors.New("no tienes permiso para compartir este apunte")
	}

	if email == nil && groupID == nil {
		return errors.New("debes proporcionar un email o un ID de grupo para compartir")
	}

	return s.repo.ShareNote(ctx, noteID, email, groupID)
}

// GetSharedNotes recupera todos los apuntes que la comunidad ha compartido con este usuario.
func (s *Service) GetSharedNotes(ctx context.Context, userID string) ([]Note, error) {
	email, err := s.repo.GetUserEmail(ctx, userID)
	if err != nil {
		return nil, errors.New("no se pudo identificar el correo del usuario")
	}
	return s.repo.GetSharedWithMe(ctx, email)
}
