package notes

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/isw2-unileon/grupo10/backend/internal/users"
)

const testAuthor = "author-1"

// newTestHandler wires the notes routes behind the real users auth middleware,
// backed by the in-memory fakeRepo, and returns a valid token for testAuthor so
// tests exercise routing + auth + handler without a database.
func newTestHandler(t *testing.T) (http.Handler, string, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	svc := NewService(repo)
	issuer := users.NewJWTIssuer("test-secret", time.Hour)

	token, err := issuer.Issue(&users.User{ID: testAuthor})
	if err != nil {
		t.Fatalf("could not issue token: %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(svc).RegisterRoutes(mux, users.RequireAuth(issuer))
	return mux, token, repo
}

func doAuth(h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCreateNote_Created(t *testing.T) {
	h, token, _ := newTestHandler(t)

	rec := doAuth(h, http.MethodPost, "/api/notes", `{"title":"T","content":"C"}`, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var note Note
	if err := json.Unmarshal(rec.Body.Bytes(), &note); err != nil {
		t.Fatalf("invalid response body: %v", err)
	}
	if note.AuthorID != testAuthor || note.Status != StatusDraft {
		t.Errorf("unexpected note: %+v", note)
	}
}

func TestCreateNote_InvalidJSON(t *testing.T) {
	h, token, _ := newTestHandler(t)

	rec := doAuth(h, http.MethodPost, "/api/notes", `{not json`, token)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateNote_Unauthorized(t *testing.T) {
	h, _, _ := newTestHandler(t)

	rec := doAuth(h, http.MethodPost, "/api/notes", `{"title":"T","content":"C"}`, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}

func TestCreateNote_RepoError(t *testing.T) {
	h, token, repo := newTestHandler(t)
	repo.failCreate = errors.New("db down")

	rec := doAuth(h, http.MethodPost, "/api/notes", `{"title":"T","content":"C"}`, token)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on repo error, got %d", rec.Code)
	}
}

func TestListNotes_OnlyMine(t *testing.T) {
	h, token, repo := newTestHandler(t)
	repo.seedNote(testAuthor, "mine", StatusDraft)
	repo.seedNote("someone-else", "theirs", StatusDraft)

	rec := doAuth(h, http.MethodGet, "/api/notes", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var list []Note
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("invalid response body: %v", err)
	}
	if len(list) != 1 || list[0].Content != "mine" {
		t.Errorf("expected only the author's note, got %+v", list)
	}
}

func TestUpdateNote_OK(t *testing.T) {
	h, token, repo := newTestHandler(t)
	id := repo.seedNote(testAuthor, "old", StatusDraft)

	rec := doAuth(h, http.MethodPut, "/api/notes/"+id, `{"title":"New","content":"updated"}`, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if repo.notes[id].Content != "updated" {
		t.Errorf("note was not updated: %+v", repo.notes[id])
	}
}

func TestDeleteNote_OK(t *testing.T) {
	h, token, repo := newTestHandler(t)
	id := repo.seedNote(testAuthor, "x", StatusDraft)

	rec := doAuth(h, http.MethodDelete, "/api/notes/"+id, "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if _, ok := repo.notes[id]; ok {
		t.Error("note was not deleted")
	}
}

func TestSubmitNote_SetsPending(t *testing.T) {
	h, token, repo := newTestHandler(t)
	id := repo.seedNote(testAuthor, "x", StatusDraft)

	rec := doAuth(h, http.MethodPost, "/api/notes/"+id+"/submit", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if repo.notes[id].Status != StatusPending {
		t.Errorf("expected pending, got %q", repo.notes[id].Status)
	}
}

func TestApproveNote_InvalidJSON(t *testing.T) {
	h, token, repo := newTestHandler(t)
	id := repo.seedNote(testAuthor, "x", StatusPending)

	rec := doAuth(h, http.MethodPost, "/api/notes/"+id+"/approve", `{bad`, token)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestApproveNote_OK(t *testing.T) {
	h, token, repo := newTestHandler(t)
	id := repo.seedNote(testAuthor, "x", StatusPending)

	rec := doAuth(h, http.MethodPost, "/api/notes/"+id+"/approve", `{"feedback":"nice"}`, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if repo.notes[id].Status != StatusApproved {
		t.Errorf("expected approved, got %q", repo.notes[id].Status)
	}
}

func TestListPending_OK(t *testing.T) {
	h, token, repo := newTestHandler(t)
	repo.seedNote(testAuthor, "draft", StatusDraft)
	repo.seedNote("other", "pending", StatusPending)

	rec := doAuth(h, http.MethodGet, "/api/teacher/notes/pending", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var list []Note
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("invalid response body: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 pending note, got %d", len(list))
	}
}

func TestShareNote_InvalidJSON(t *testing.T) {
	h, token, repo := newTestHandler(t)
	id := repo.seedNote(testAuthor, "x", StatusApproved)

	rec := doAuth(h, http.MethodPost, "/api/notes/"+id+"/share", `{bad`, token)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestShareNote_OK(t *testing.T) {
	h, token, repo := newTestHandler(t)
	id := repo.seedNote(testAuthor, "x", StatusApproved)

	rec := doAuth(h, http.MethodPost, "/api/notes/"+id+"/share", `{"email":"friend@uni.es"}`, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if repo.lastShareEmail == nil || *repo.lastShareEmail != "friend@uni.es" {
		t.Errorf("expected note shared with friend@uni.es, got %v", repo.lastShareEmail)
	}
}

// buildDocx returns the bytes of a minimal valid .docx (a zip containing
// word/document.xml) holding the given paragraphs.
func buildDocx(t *testing.T, paragraphs ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("could not create docx entry: %v", err)
	}
	var body strings.Builder
	body.WriteString(`<?xml version="1.0"?><w:document xmlns:w="x"><w:body>`)
	for _, p := range paragraphs {
		body.WriteString(`<w:p><w:r><w:t>` + p + `</w:t></w:r></w:p>`)
	}
	body.WriteString(`</w:body></w:document>`)
	if _, err := f.Write([]byte(body.String())); err != nil {
		t.Fatalf("could not write docx body: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("could not close docx zip: %v", err)
	}
	return buf.Bytes()
}

// uploadRequest builds a multipart upload request for the given filename/content.
func uploadRequest(t *testing.T, token, filename string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("title", "My Doc")
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("could not create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("could not write form file: %v", err)
	}
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/notes/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestUploadNote_DocxCreated(t *testing.T) {
	h, token, _ := newTestHandler(t)
	docx := buildDocx(t, "Primer parrafo", "Segundo parrafo")

	req := uploadRequest(t, token, "apuntes.docx", docx)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var note Note
	if err := json.Unmarshal(rec.Body.Bytes(), &note); err != nil {
		t.Fatalf("invalid response body: %v", err)
	}
	if !strings.Contains(note.Content, "Primer parrafo") || !strings.Contains(note.Content, "Segundo parrafo") {
		t.Errorf("docx text was not extracted: %q", note.Content)
	}
}

func TestUploadNote_RejectsNonDocx(t *testing.T) {
	h, token, _ := newTestHandler(t)

	req := uploadRequest(t, token, "notes.pdf", []byte("not a docx"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-docx, got %d", rec.Code)
	}
}

func TestUploadNote_RejectsCorruptDocx(t *testing.T) {
	h, token, _ := newTestHandler(t)

	req := uploadRequest(t, token, "broken.docx", []byte("this is not a zip"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for corrupt docx, got %d", rec.Code)
	}
}

func TestListSharedNotes_OK(t *testing.T) {
	h, token, repo := newTestHandler(t)
	repo.emails[testAuthor] = "me@uni.es"
	repo.shared["me@uni.es"] = []Note{{ID: "n1", Title: "shared with me"}}

	rec := doAuth(h, http.MethodGet, "/api/notes/shared", "", token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var list []Note
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("invalid response body: %v", err)
	}
	if len(list) != 1 || list[0].ID != "n1" {
		t.Errorf("expected the shared note, got %+v", list)
	}
}
