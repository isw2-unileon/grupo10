package groups

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

const (
	// defaultGroqURL is the Groq chat-completions endpoint (OpenAI-compatible).
	defaultGroqURL = "https://api.groq.com/openai/v1/chat/completions"
	groqModel      = "llama-3.3-70b-versatile"
	maxAIQuestions = 30
)

// QuizAIParams describes a request to generate a quiz from teaching material.
type QuizAIParams struct {
	Content      string
	Difficulty   string
	NumQuestions int
	Focus        string
}

// --- Groq (OpenAI-compatible) wire types ---

type aiChatRequest struct {
	Model          string          `json:"model"`
	Messages       []aiChatMessage `json:"messages"`
	ResponseFormat *aiResponseFmt  `json:"response_format,omitempty"`
	Temperature    float64         `json:"temperature"`
}

type aiResponseFmt struct {
	Type string `json:"type"`
}

type aiChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiChatResponse struct {
	Choices []struct {
		Message aiChatMessage `json:"message"`
	} `json:"choices"`
}

// aiQuizPayload is the JSON shape the model must return (and that we send back
// when asking it to improve an existing quiz).
type aiQuizPayload struct {
	Questions []aiQuizQuestion `json:"questions"`
}

type aiQuizQuestion struct {
	QuestionText string         `json:"question_text"`
	Options      []aiQuizOption `json:"options"`
}

type aiQuizOption struct {
	OptionText string `json:"option_text"`
	IsCorrect  bool   `json:"is_correct"`
}

// GenerateQuiz asks the AI to build a multiple-choice quiz from the given
// teaching material. It does NOT persist anything: it returns the questions so
// the caller (a teacher) can review and edit them before saving through the
// normal create-quiz flow.
func (s *Service) GenerateQuiz(ctx context.Context, p QuizAIParams) ([]QuizQuestion, error) {
	content := strings.TrimSpace(p.Content)
	if content == "" {
		return nil, errors.New("el material está vacío: sube un documento o pega texto")
	}
	if p.NumQuestions <= 0 {
		p.NumQuestions = 5
	}
	if p.NumQuestions > maxAIQuestions {
		p.NumQuestions = maxAIQuestions
	}
	difficulty := strings.TrimSpace(p.Difficulty)
	if difficulty == "" {
		difficulty = "media"
	}

	system := "You are an exam author integrated into Learning Platform, a university LMS. You " +
		"create multiple-choice quizzes from the teaching material a teacher provides.\n" +
		"Rules:\n" +
		"- Each question MUST have exactly 4 options and EXACTLY ONE correct option.\n" +
		"- Base every question strictly on the provided material; never invent facts it does not support.\n" +
		"- Respect the requested difficulty and focus.\n" +
		"- Write all questions and options in Spanish, regardless of the language of these instructions.\n" +
		"Return ONLY a JSON object with this exact shape: " +
		`{"questions":[{"question_text":"...","options":[{"option_text":"...","is_correct":true},` +
		`{"option_text":"...","is_correct":false},{"option_text":"...","is_correct":false},` +
		`{"option_text":"...","is_correct":false}]}]}`

	var b strings.Builder
	fmt.Fprintf(&b, "Generate %d multiple-choice questions.\n", p.NumQuestions)
	fmt.Fprintf(&b, "Difficulty: %s.\n", difficulty)
	if f := strings.TrimSpace(p.Focus); f != "" {
		fmt.Fprintf(&b, "Focus especially on: %s.\n", f)
	}
	b.WriteString("\nTeaching material:\n")
	b.WriteString(content)

	payload, err := s.callQuizAI(ctx, system, b.String())
	if err != nil {
		return nil, err
	}
	return payload.toQuestions(), nil
}

// ImproveQuiz refines an existing (not yet saved) set of questions according to
// a free-text instruction (e.g. "raise the difficulty", "level up", "rephrase
// question 2"). It returns the improved questions without persisting them.
func (s *Service) ImproveQuiz(ctx context.Context, questions []QuizQuestion, instruction string) ([]QuizQuestion, error) {
	if len(questions) == 0 {
		return nil, errors.New("no hay preguntas que mejorar")
	}
	instruction = strings.TrimSpace(instruction)
	if instruction == "" {
		instruction = "Mejora la redacción y la calidad de las preguntas manteniendo el tema."
	}

	current, err := json.Marshal(toPayload(questions))
	if err != nil {
		return nil, fmt.Errorf("error empaquetando las preguntas: %w", err)
	}

	system := "You are an exam author integrated into Learning Platform. You receive an existing " +
		"multiple-choice quiz as JSON plus an instruction, and you return an IMPROVED quiz.\n" +
		"Rules:\n" +
		"- Keep the same number of questions unless the instruction says otherwise.\n" +
		"- Each question keeps exactly 4 options with EXACTLY ONE correct option.\n" +
		"- Apply the instruction (raise difficulty, level up, rephrase, fix a specific question...).\n" +
		"- Write everything in Spanish.\n" +
		"Return ONLY a JSON object with the SAME shape you received."

	user := "Instruction: " + instruction + "\n\nCurrent quiz JSON:\n" + string(current)

	payload, err := s.callQuizAI(ctx, system, user)
	if err != nil {
		return nil, err
	}
	return payload.toQuestions(), nil
}

// callQuizAI performs the Groq request and parses the JSON quiz it returns.
func (s *Service) callQuizAI(ctx context.Context, system, user string) (*aiQuizPayload, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return nil, errors.New("no hay clave de API configurada para la IA (Groq)")
	}

	reqBody := aiChatRequest{
		Model: groqModel,
		Messages: []aiChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		ResponseFormat: &aiResponseFmt{Type: "json_object"},
		Temperature:    0.4,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error empaquetando JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, defaultGroqURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("error creando la petición HTTP: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error conectando con la IA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("la IA devolvió un error (%d): %s", resp.StatusCode, string(body))
	}

	var chat aiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chat); err != nil {
		return nil, errors.New("error leyendo la respuesta de la IA")
	}
	if len(chat.Choices) == 0 {
		return nil, errors.New("la IA no devolvió contenido")
	}

	var payload aiQuizPayload
	if err := json.Unmarshal([]byte(chat.Choices[0].Message.Content), &payload); err != nil {
		return nil, errors.New("la IA devolvió un formato inesperado")
	}
	if len(payload.Questions) == 0 {
		return nil, errors.New("la IA no generó preguntas")
	}
	return &payload, nil
}

// toQuestions converts the AI payload into the domain QuizQuestion slice used by
// the manual create-quiz flow.
func (p *aiQuizPayload) toQuestions() []QuizQuestion {
	out := make([]QuizQuestion, 0, len(p.Questions))
	for i, q := range p.Questions {
		question := QuizQuestion{QuestionText: q.QuestionText, Position: i}
		for _, o := range q.Options {
			question.Options = append(question.Options, QuizOption{
				OptionText: o.OptionText,
				IsCorrect:  o.IsCorrect,
			})
		}
		out = append(out, question)
	}
	return out
}

// toPayload is the inverse of toQuestions, used to feed the current quiz back to
// the model when asking it to improve the questions.
func toPayload(questions []QuizQuestion) aiQuizPayload {
	p := aiQuizPayload{Questions: make([]aiQuizQuestion, 0, len(questions))}
	for _, q := range questions {
		aq := aiQuizQuestion{QuestionText: q.QuestionText}
		for _, o := range q.Options {
			aq.Options = append(aq.Options, aiQuizOption{
				OptionText: o.OptionText,
				IsCorrect:  o.IsCorrect,
			})
		}
		p.Questions = append(p.Questions, aq)
	}
	return p
}
