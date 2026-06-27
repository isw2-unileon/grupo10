package aitutor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeRepo is an in-memory Repository used to test the service without a
// database. It returns canned section titles and failed-question context and
// can inject errors.
type fakeRepo struct {
	titles []string
	fallos []FailedQuestionContext

	failTitles error
	failFallos error

	// Captured inputs for assertions.
	lastStudentID  string
	lastSectionIDs []string
}

func (f *fakeRepo) GetSectionsTitles(_ context.Context, sectionIDs []string) ([]string, error) {
	f.lastSectionIDs = sectionIDs
	if f.failTitles != nil {
		return nil, f.failTitles
	}
	return f.titles, nil
}

func (f *fakeRepo) GetFailedQuestionsContext(_ context.Context, studentID string, _ []string) ([]FailedQuestionContext, error) {
	f.lastStudentID = studentID
	if f.failFallos != nil {
		return nil, f.failFallos
	}
	return f.fallos, nil
}

// validQuizJSON is a well-formed AI answer with the structure the service expects.
const validQuizJSON = `{"questions":[
	{"question_text":"¿Qué es un puntero?","options":[
		{"text":"Una dirección de memoria","is_correct":true},
		{"text":"Un bucle","is_correct":false},
		{"text":"Un tipo de error","is_correct":false},
		{"text":"Una palabra clave","is_correct":false}],
	 "explanation":"Un puntero guarda una dirección."}
]}`

// fakeGroq returns a test server that asserts the request shape and replies with
// the given content as the assistant message. It points the service at itself.
func fakeGroq(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("expected bearer auth header, got %q", got)
		}
		var body groqRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("invalid request body: %v", err)
		}
		if len(body.Messages) != 2 || body.Messages[0].Role != "system" || body.Messages[1].Role != "user" {
			t.Errorf("unexpected messages: %+v", body.Messages)
		}
		if body.ResponseFormat == nil || body.ResponseFormat.Type != "json_object" {
			t.Errorf("expected JSON mode to be requested, got %+v", body.ResponseFormat)
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": content}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestGenerateAIQuiz_NoSections(t *testing.T) {
	svc := NewService(&fakeRepo{titles: nil})

	_, err := svc.GenerateAIQuiz(context.Background(), "student-1", []string{"s1"})
	if err == nil {
		t.Fatal("expected error when no valid section titles are found")
	}
}

func TestGenerateAIQuiz_TitlesRepoError(t *testing.T) {
	svc := NewService(&fakeRepo{failTitles: errors.New("db down")})

	_, err := svc.GenerateAIQuiz(context.Background(), "student-1", []string{"s1"})
	if err == nil {
		t.Fatal("expected error when the repository fails to load titles")
	}
}

func TestGenerateAIQuiz_NoAPIKey(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "")
	svc := NewService(&fakeRepo{titles: []string{"Punteros"}})

	_, err := svc.GenerateAIQuiz(context.Background(), "student-1", []string{"s1"})
	if err == nil {
		t.Fatal("expected error when GROQ_API_KEY is unset")
	}
}

func TestGenerateAIQuiz_HappyPath(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-key")
	repo := &fakeRepo{titles: []string{"Punteros", "Slices"}}
	srv := fakeGroq(t, validQuizJSON)
	defer srv.Close()

	svc := NewService(repo)
	svc.aiURL = srv.URL

	quiz, err := svc.GenerateAIQuiz(context.Background(), "student-1", []string{"s1", "s2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(quiz.Questions) != 1 || quiz.Questions[0].QuestionText != "¿Qué es un puntero?" {
		t.Errorf("AI questions were not parsed: %+v", quiz.Questions)
	}
	if quiz.SectionID != "s1,s2" {
		t.Errorf("expected section IDs joined, got %q", quiz.SectionID)
	}
	if !strings.Contains(quiz.SectionTitle, "2 Bloques") {
		t.Errorf("expected the title to mention the number of blocks, got %q", quiz.SectionTitle)
	}
	if repo.lastStudentID != "student-1" {
		t.Errorf("expected the student ID forwarded to the repo, got %q", repo.lastStudentID)
	}
}

func TestGenerateAIQuiz_IncludesFailedQuestionsInPrompt(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-key")
	repo := &fakeRepo{
		titles: []string{"Punteros"},
		fallos: []FailedQuestionContext{
			{QuestionText: "¿Qué hace nil?", WrongAnswer: "nada", RightAnswer: "puntero vacío"},
		},
	}

	var capturedUserPrompt string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body groqRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.Messages) == 2 {
			capturedUserPrompt = body.Messages[1].Content
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":` + jsonString(validQuizJSON) + `}}]}`))
	}))
	defer srv.Close()

	svc := NewService(repo)
	svc.aiURL = srv.URL

	if _, err := svc.GenerateAIQuiz(context.Background(), "student-1", []string{"s1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedUserPrompt, "¿Qué hace nil?") {
		t.Errorf("expected the failed question to be injected into the prompt, got %q", capturedUserPrompt)
	}
}

func TestGenerateAIQuiz_StripsMarkdownFences(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-key")
	repo := &fakeRepo{titles: []string{"Punteros"}}
	fenced := "```json\n" + validQuizJSON + "\n```"
	srv := fakeGroq(t, fenced)
	defer srv.Close()

	svc := NewService(repo)
	svc.aiURL = srv.URL

	quiz, err := svc.GenerateAIQuiz(context.Background(), "student-1", []string{"s1"})
	if err != nil {
		t.Fatalf("unexpected error parsing markdown-fenced JSON: %v", err)
	}
	if len(quiz.Questions) != 1 {
		t.Errorf("expected 1 question after stripping fences, got %d", len(quiz.Questions))
	}
}

func TestGenerateAIQuiz_Non200(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-key")
	repo := &fakeRepo{titles: []string{"Punteros"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	svc := NewService(repo)
	svc.aiURL = srv.URL

	if _, err := svc.GenerateAIQuiz(context.Background(), "student-1", []string{"s1"}); err == nil {
		t.Fatal("expected error when the AI returns a non-200 status")
	}
}

func TestGenerateAIQuiz_EmptyChoices(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-key")
	repo := &fakeRepo{titles: []string{"Punteros"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	svc := NewService(repo)
	svc.aiURL = srv.URL

	if _, err := svc.GenerateAIQuiz(context.Background(), "student-1", []string{"s1"}); err == nil {
		t.Fatal("expected error when the AI returns no choices")
	}
}

func TestGenerateAIQuiz_InvalidAIJSON(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-key")
	repo := &fakeRepo{titles: []string{"Punteros"}}
	srv := fakeGroq(t, "esto no es JSON")
	defer srv.Close()

	svc := NewService(repo)
	svc.aiURL = srv.URL

	if _, err := svc.GenerateAIQuiz(context.Background(), "student-1", []string{"s1"}); err == nil {
		t.Fatal("expected error when the AI answer is not valid JSON")
	}
}

// jsonString encodes s as a JSON string literal (with surrounding quotes) so it
// can be embedded inside a hand-written JSON payload.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
