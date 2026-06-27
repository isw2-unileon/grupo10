package aitutor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/isw2-unileon/grupo10/backend/internal/users"
)

const testStudent = "student-1"

// newTestHandler wires the ai-quiz route behind the real users auth middleware,
// backed by an in-memory repo and a fake Groq server, and returns a valid token
// so tests exercise routing + auth + handler without a database or real AI.
func newTestHandler(t *testing.T, repo *fakeRepo, aiURL string) (http.Handler, string) {
	t.Helper()
	svc := NewService(repo)
	if aiURL != "" {
		svc.aiURL = aiURL
	}
	issuer := users.NewJWTIssuer("test-secret", time.Hour)

	token, err := issuer.Issue(&users.User{ID: testStudent})
	if err != nil {
		t.Fatalf("could not issue token: %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(svc, issuer).RegisterRoutes(mux)
	return mux, token
}

func doReq(h http.Handler, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestGenerateQuiz_Unauthorized(t *testing.T) {
	h, _ := newTestHandler(t, &fakeRepo{}, "")

	rec := doReq(h, "/api/ai-quiz?sections=s1", "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}

func TestGenerateQuiz_MissingSections(t *testing.T) {
	h, token := newTestHandler(t, &fakeRepo{}, "")

	rec := doReq(h, "/api/ai-quiz", token)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when no sections are given, got %d", rec.Code)
	}
}

func TestGenerateQuiz_OK(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-key")
	repo := &fakeRepo{titles: []string{"Punteros"}}
	srv := fakeGroq(t, validQuizJSON)
	defer srv.Close()

	h, token := newTestHandler(t, repo, srv.URL)

	rec := doReq(h, "/api/ai-quiz?sections=s1,s2", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var quiz GeneratedAIQuiz
	if err := json.Unmarshal(rec.Body.Bytes(), &quiz); err != nil {
		t.Fatalf("invalid response body: %v", err)
	}
	if len(quiz.Questions) != 1 || quiz.SectionID != "s1,s2" {
		t.Errorf("unexpected quiz: %+v", quiz)
	}
}

func TestGenerateQuiz_AIError(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-key")
	repo := &fakeRepo{titles: []string{"Punteros"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	h, token := newTestHandler(t, repo, srv.URL)

	rec := doReq(h, "/api/ai-quiz?sections=s1", token)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when the AI fails, got %d", rec.Code)
	}
}
