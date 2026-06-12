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

// Service contiene la lógica de negocio de los apuntes
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

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

func (s *Service) GetNotesByAuthor(ctx context.Context, authorID string) ([]Note, error) {
	return s.repo.GetByAuthor(ctx, authorID)
}

func (s *Service) UpdateNote(ctx context.Context, id, authorID, title, content string) error {
	n := &Note{ID: id, AuthorID: authorID, Title: title, Content: content}
	return s.repo.UpdateNote(ctx, n)
}

func (s *Service) DeleteNote(ctx context.Context, id, authorID string) error {
	return s.repo.DeleteNote(ctx, id, authorID)
}

// ==========================================
// ESTRUCTURAS PARA LA API DE LA IA (GROQ/OPENAI)
// ==========================================

// Groq usa exactamente el mismo formato JSON que OpenAI, así que reutilizamos las estructuras
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

// ==========================================
// LÓGICA DE REVISIÓN CON IA
// ==========================================

func (s *Service) RequestAIReview(ctx context.Context, noteID, content string) error {
	// 1. Leemos nuestra clave gratuita de Groq
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return errors.New("no hay clave de API configurada para la IA (Groq)")
	}

	// 2. El Prompt: Qué queremos que haga la IA
	prompt := "Eres un profesor estricto pero amable. Corrige estos apuntes, señala errores ortográficos, " +
		"conceptuales y sugiere mejoras. Sé conciso y devuelve el texto bien formateado. Apuntes a corregir:\n\n" + content

	// 3. Montamos la petición
	reqBody := openAIRequest{
		Model: "llama-3.3-70b-versatile", // Modelo de Meta: Rápido, ligero y gratuito en Groq
		Messages: []openAIMsg{
			{Role: "system", Content: "Eres un tutor educativo de alto nivel que corrige apuntes. Responde siempre en español."},
			{Role: "user", Content: prompt},
		},
	}

	jsonData, _ := json.Marshal(reqBody)

	// 4. Llamamos a la API de Groq
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 5. Enviamos la petición
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

	// 6. Procesamos la respuesta
	var aiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return errors.New("error leyendo respuesta de la IA")
	}

	if len(aiResp.Choices) == 0 {
		return errors.New("la IA no devolvió ningún texto")
	}

	feedback := aiResp.Choices[0].Message.Content

	// 7. Guardamos en Postgres
	log := &AIFeedbackLog{NoteID: noteID, PromptUsed: prompt, Response: feedback}
	return s.repo.UpdateNoteWithAI(ctx, noteID, feedback, log)
}

func (s *Service) SubmitForApproval(ctx context.Context, noteID string) error {
	return s.repo.UpdateStatus(ctx, noteID, StatusPending)
}

func (s *Service) GetPendingForTeacher(ctx context.Context) ([]Note, error) {
	return s.repo.GetPending(ctx)
}

func (s *Service) ApproveNoteWithFeedback(ctx context.Context, noteID, feedback string) error {
	return s.repo.ApproveNoteWithFeedback(ctx, noteID, feedback)
}
