package users

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestHandler builds a Handler backed by the in-memory service helpers from
// service_test.go, plus the mux with the routes wired, so tests exercise the
// real method-aware routing without a database.
func newTestHandler() http.Handler {
	mux := http.NewServeMux()
	NewHandler(newTestService()).RegisterRoutes(mux)
	return mux
}

// do sends a request with the given JSON body and returns the recorded response.
func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func registerBody(in RegisterInput) string {
	b, _ := json.Marshal(map[string]string{
		"name":     in.Name,
		"email":    in.Email,
		"password": in.Password,
		"role":     in.Role,
	})
	return string(b)
}

func TestHandlerRegister_Created(t *testing.T) {
	h := newTestHandler()

	rec := do(h, http.MethodPost, "/api/register", registerBody(validInput()))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp authResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected a token in the response")
	}
	if resp.User == nil || resp.User.Email != "ada@example.com" {
		t.Errorf("expected the created user, got %+v", resp.User)
	}
	// The password hash must never be exposed in the JSON response.
	if strings.Contains(rec.Body.String(), "password_hash") ||
		strings.Contains(rec.Body.String(), "supersecret") {
		t.Error("response must not leak the password or its hash")
	}
}

func TestHandlerRegister_DuplicateEmailConflict(t *testing.T) {
	h := newTestHandler()
	body := registerBody(validInput())

	if rec := do(h, http.MethodPost, "/api/register", body); rec.Code != http.StatusCreated {
		t.Fatalf("first register should succeed, got %d", rec.Code)
	}

	rec := do(h, http.MethodPost, "/api/register", body)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 on duplicate email, got %d", rec.Code)
	}
}

func TestHandlerRegister_InvalidRoleBadRequest(t *testing.T) {
	h := newTestHandler()
	in := validInput()
	in.Role = "admin"

	rec := do(h, http.MethodPost, "/api/register", registerBody(in))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on invalid role, got %d", rec.Code)
	}
}

func TestHandlerRegister_ValidationBadRequest(t *testing.T) {
	h := newTestHandler()
	in := validInput()
	in.Password = "short"

	rec := do(h, http.MethodPost, "/api/register", registerBody(in))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on validation failure, got %d", rec.Code)
	}
}

func TestHandlerRegister_InvalidJSONBadRequest(t *testing.T) {
	h := newTestHandler()

	rec := do(h, http.MethodPost, "/api/register", "{not-json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on malformed JSON, got %d", rec.Code)
	}
}

func TestHandlerLogin_OK(t *testing.T) {
	h := newTestHandler()
	if rec := do(h, http.MethodPost, "/api/register", registerBody(validInput())); rec.Code != http.StatusCreated {
		t.Fatalf("setup register failed: %d", rec.Code)
	}

	body, _ := json.Marshal(loginRequest{Email: "ada@example.com", Password: "supersecret"})
	rec := do(h, http.MethodPost, "/api/login", string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp authResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected a token on successful login")
	}
}

func TestHandlerLogin_WrongPasswordUnauthorized(t *testing.T) {
	h := newTestHandler()
	if rec := do(h, http.MethodPost, "/api/register", registerBody(validInput())); rec.Code != http.StatusCreated {
		t.Fatalf("setup register failed: %d", rec.Code)
	}

	body, _ := json.Marshal(loginRequest{Email: "ada@example.com", Password: "wrongpass"})
	rec := do(h, http.MethodPost, "/api/login", string(body))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on wrong password, got %d", rec.Code)
	}
}

func TestHandlerLogin_UnknownEmailUnauthorized(t *testing.T) {
	h := newTestHandler()

	body, _ := json.Marshal(loginRequest{Email: "nobody@example.com", Password: "whatever"})
	rec := do(h, http.MethodPost, "/api/login", string(body))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on unknown email, got %d", rec.Code)
	}
}

func TestHandlerLogin_InvalidJSONBadRequest(t *testing.T) {
	h := newTestHandler()

	rec := do(h, http.MethodPost, "/api/login", "{not-json")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on malformed JSON, got %d", rec.Code)
	}
}
