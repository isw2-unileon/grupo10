package aitutor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Service implementa la lógica de negocio.
type Service struct {
	repo Repository
}

// NewService inicializa el servicio de IA.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Estructuras de red internas para Groq (que usa el mismo formato que OpenAI)
type groqMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqRequest struct {
	Model          string          `json:"model"`
	Messages       []groqMsg       `json:"messages"`
	ResponseFormat *groqRespFormat `json:"response_format,omitempty"`
	Temperature    float64         `json:"temperature"`
}

type groqRespFormat struct {
	Type string `json:"type"`
}

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// GenerateAIQuiz solicita el cuestionario multitemático a Llama 3.3 vía Groq.
func (s *Service) GenerateAIQuiz(ctx context.Context, studentID string, sectionIDs []string) (*GeneratedAIQuiz, error) {
	titles, err := s.repo.GetSectionsTitles(ctx, sectionIDs)
	if err != nil || len(titles) == 0 {
		return nil, fmt.Errorf("no se encontraron temas válidos")
	}

	fallos, _ := s.repo.GetFailedQuestionsContext(ctx, studentID, sectionIDs)
	contextoTemas := strings.Join(titles, ", ")

	// 1. CONSTRUCCIÓN DEL PROMPT PEDAGÓGICO
	promptContext := "El alumno no tiene errores de antemano registrados. Haz preguntas de nivel básico/medio sobre el temario para comprobar sus conocimientos generales."
	if len(fallos) > 0 {
		//nolint:misspell
		promptContext = "ATENCIÓN: El alumno ha fallado recientemente en estos conceptos. Crea preguntas MUY SIMILARES (del mismo nivel y temática) para reforzar su debilidad:\n"
		for _, f := range fallos {
			promptContext += fmt.Sprintf("- Falló en la pregunta: '%s'. (Eligió erróneamente '%s', pero la respuesta correcta era '%s').\n", f.QuestionText, f.WrongAnswer, f.RightAnswer)
		}
	}

	//nolint:misspell
	systemPrompt := `Eres un profesor universitario y un tutor interactivo. Tu objetivo es generar un test de refuerzo.
DEBES responder ÚNICA Y EXCLUSIVAMENTE con un objeto JSON válido, sin bloques de código Markdown, sin textos introductorios y sin saludos.
La estructura estricta del JSON debe ser:
{
  "questions": [
    {
      "question_text": "Texto de la pregunta",
      "options": [
        {"text": "Opción correcta", "is_correct": true},
        {"text": "Falsa 1", "is_correct": false},
        {"text": "Falsa 2", "is_correct": false},
        {"text": "Falsa 3", "is_correct": false}
      ],
      "explanation": "Explicación breve de por qué es la correcta."
    }
  ]
}`

	userPrompt := fmt.Sprintf("Genera un cuestionario de exactamente 3 preguntas que mezcle estos temas: %s.\n\nContexto del alumno:\n%s", contextoTemas, promptContext)

	// 2. RECUPERAR LA API KEY DE GROQ DEL ENTORNO
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return nil, errors.New("no hay clave de API configurada para la IA (Falta GROQ_API_KEY)")
	}

	// 3. PETICIÓN A LA API DE GROQ
	reqBody := groqRequest{
		Model: "llama-3.3-70b-versatile", // El mismo que usáis para apuntes
		Messages: []groqMsg{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &groqRespFormat{Type: "json_object"}, // Forzamos JSON Mode
		Temperature:    0.7,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error al empaquetar JSON: %w", err)
	}

	// Endpoint estándar de Groq
	groqURL := "https://api.groq.com/openai/v1/chat/completions"

	req, err := http.NewRequestWithContext(ctx, "POST", groqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error al crear petición HTTP: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error conectando con Groq: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("la IA devolvió un error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var aiResp groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return nil, errors.New("error leyendo respuesta de Groq")
	}

	if len(aiResp.Choices) == 0 {
		return nil, errors.New("la IA no devolvió ningún texto")
	}

	jsonIA := aiResp.Choices[0].Message.Content

	// 4. LIMPIEZA DE MARKDOWN (Defensa contra alucinaciones de Llama)
	jsonIA = strings.TrimSpace(jsonIA)
	if strings.HasPrefix(jsonIA, "```json") {
		jsonIA = strings.TrimPrefix(jsonIA, "```json")
		jsonIA = strings.TrimSuffix(jsonIA, "```")
		jsonIA = strings.TrimSpace(jsonIA)
	} else if strings.HasPrefix(jsonIA, "```") {
		jsonIA = strings.TrimPrefix(jsonIA, "```")
		jsonIA = strings.TrimSuffix(jsonIA, "```")
		jsonIA = strings.TrimSpace(jsonIA)
	}

	// 5. MAPEO AL STRUCT DE GO
	var finalQuiz GeneratedAIQuiz
	if err := json.Unmarshal([]byte(jsonIA), &finalQuiz); err != nil {
		return nil, fmt.Errorf("fallo al parsear el JSON de la IA: %w", err)
	}

	// Completamos los metadatos visuales
	finalQuiz.SectionID = strings.Join(sectionIDs, ",")
	finalQuiz.SectionTitle = fmt.Sprintf("Cuestionario IA: Refuerzo Adaptativo (%d Bloques)", len(titles))

	return &finalQuiz, nil
}
