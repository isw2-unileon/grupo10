package notes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeRepo is an in-memory Repository used to test the service and handlers
// without a database. It records the relevant calls so tests can assert on them.
type fakeRepo struct {
	notes  map[string]*Note  // by note ID
	emails map[string]string // userID -> email
	shared map[string][]Note // recipient email -> notes shared with them
	nextID int

	// Captured side effects for assertions.
	lastAIFeedback string
	lastAILog      *AIFeedbackLog
	lastShareEmail *string
	lastShareGroup *string

	// Error injection: when set, the matching method returns this error.
	failCreate  error
	failGetByID error
	failShare   error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		notes:  make(map[string]*Note),
		emails: make(map[string]string),
		shared: make(map[string][]Note),
	}
}

func (f *fakeRepo) CreateNote(_ context.Context, n *Note) error {
	if f.failCreate != nil {
		return f.failCreate
	}
	f.nextID++
	n.ID = "note-" + string(rune('0'+f.nextID))
	f.notes[n.ID] = n
	return nil
}

func (f *fakeRepo) GetByAuthor(_ context.Context, authorID string) ([]Note, error) {
	var list []Note
	for _, n := range f.notes {
		if n.AuthorID == authorID {
			list = append(list, *n)
		}
	}
	return list, nil
}

func (f *fakeRepo) GetByID(_ context.Context, id string) (*Note, error) {
	if f.failGetByID != nil {
		return nil, f.failGetByID
	}
	n, ok := f.notes[id]
	if !ok {
		return nil, ErrNoteNotFound
	}
	return n, nil
}

func (f *fakeRepo) UpdateNote(_ context.Context, n *Note) error {
	existing, ok := f.notes[n.ID]
	if !ok || existing.AuthorID != n.AuthorID {
		return ErrNoteNotFound
	}
	existing.Title = n.Title
	existing.Content = n.Content
	return nil
}

func (f *fakeRepo) UpdateStatus(_ context.Context, noteID string, status NoteStatus) error {
	n, ok := f.notes[noteID]
	if !ok {
		return ErrNoteNotFound
	}
	n.Status = status
	return nil
}

func (f *fakeRepo) UpdateNoteWithAI(_ context.Context, noteID, feedback string, log *AIFeedbackLog) error {
	n, ok := f.notes[noteID]
	if !ok {
		return ErrNoteNotFound
	}
	n.Status = StatusAiReviewed
	n.AiFeedback = &feedback
	f.lastAIFeedback = feedback
	f.lastAILog = log
	return nil
}

func (f *fakeRepo) DeleteNote(_ context.Context, id, authorID string) error {
	n, ok := f.notes[id]
	if !ok || n.AuthorID != authorID {
		return ErrNoteNotFound
	}
	delete(f.notes, id)
	return nil
}

func (f *fakeRepo) GetPending(_ context.Context) ([]Note, error) {
	var list []Note
	for _, n := range f.notes {
		if n.Status == StatusPending {
			list = append(list, *n)
		}
	}
	return list, nil
}

func (f *fakeRepo) ApproveNoteWithFeedback(_ context.Context, noteID, feedback string) error {
	n, ok := f.notes[noteID]
	if !ok {
		return ErrNoteNotFound
	}
	n.Status = StatusApproved
	n.TeacherFeedback = &feedback
	return nil
}

func (f *fakeRepo) GetUserEmail(_ context.Context, userID string) (string, error) {
	email, ok := f.emails[userID]
	if !ok {
		return "", ErrNoteNotFound
	}
	return email, nil
}

func (f *fakeRepo) ShareNote(_ context.Context, _ string, email, groupID *string) error {
	if f.failShare != nil {
		return f.failShare
	}
	f.lastShareEmail = email
	f.lastShareGroup = groupID
	return nil
}

func (f *fakeRepo) GetSharedWithMe(_ context.Context, email string) ([]Note, error) {
	return f.shared[email], nil
}

// seedNote inserts a note owned by authorID and returns its generated ID.
func (f *fakeRepo) seedNote(authorID, content string, status NoteStatus) string {
	f.nextID++
	id := "note-" + string(rune('0'+f.nextID))
	f.notes[id] = &Note{ID: id, AuthorID: authorID, Content: content, Status: status}
	return id
}

func TestCreateNote_StartsAsDraft(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	note, err := svc.CreateNote(context.Background(), "author-1", "Title", "Body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if note.Status != StatusDraft {
		t.Errorf("expected status %q, got %q", StatusDraft, note.Status)
	}
	if note.AuthorID != "author-1" {
		t.Errorf("expected author-1, got %q", note.AuthorID)
	}
	if note.SubjectID != nil {
		t.Errorf("expected nil subject, got %v", *note.SubjectID)
	}
}

func TestSubmitForApproval_SetsPending(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	id := repo.seedNote("author-1", "x", StatusDraft)

	if err := svc.SubmitForApproval(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := repo.notes[id].Status; got != StatusPending {
		t.Errorf("expected status pending, got %q", got)
	}
}

func TestGetPendingForTeacher_OnlyPending(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	repo.seedNote("author-1", "draft", StatusDraft)
	repo.seedNote("author-2", "pending", StatusPending)

	pending, err := svc.GetPendingForTeacher(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pending) != 1 || pending[0].Status != StatusPending {
		t.Errorf("expected 1 pending note, got %+v", pending)
	}
}

func TestApproveNoteWithFeedback_Approves(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	id := repo.seedNote("author-1", "x", StatusPending)

	if err := svc.ApproveNoteWithFeedback(context.Background(), id, "great job"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	n := repo.notes[id]
	if n.Status != StatusApproved {
		t.Errorf("expected approved, got %q", n.Status)
	}
	if n.TeacherFeedback == nil || *n.TeacherFeedback != "great job" {
		t.Errorf("teacher feedback not stored: %v", n.TeacherFeedback)
	}
}

func TestShareNote_RejectsNonOwner(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	id := repo.seedNote("owner", "x", StatusApproved)

	email := "friend@uni.es"
	err := svc.ShareNote(context.Background(), id, "intruder", &email, nil)
	if err == nil {
		t.Fatal("expected permission error sharing someone else's note")
	}
}

func TestShareNote_RequiresEmailOrGroup(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	id := repo.seedNote("owner", "x", StatusApproved)

	err := svc.ShareNote(context.Background(), id, "owner", nil, nil)
	if err == nil {
		t.Fatal("expected error when neither email nor group is provided")
	}
}

func TestShareNote_OwnerWithEmail(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	id := repo.seedNote("owner", "x", StatusApproved)

	email := "friend@uni.es"
	if err := svc.ShareNote(context.Background(), id, "owner", &email, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastShareEmail == nil || *repo.lastShareEmail != email {
		t.Errorf("expected note shared with %q, got %v", email, repo.lastShareEmail)
	}
}

func TestGetSharedNotes_UsesUserEmail(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	repo.emails["user-1"] = "me@uni.es"
	repo.shared["me@uni.es"] = []Note{{ID: "n1", Title: "shared"}}

	notes, err := svc.GetSharedNotes(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 1 || notes[0].ID != "n1" {
		t.Errorf("expected the note shared with me@uni.es, got %+v", notes)
	}
}

func TestRequestAIReview_NoAPIKey(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "")
	repo := newFakeRepo()
	svc := NewService(repo)

	err := svc.RequestAIReview(context.Background(), "note-1", "content")
	if err == nil {
		t.Fatal("expected error when GROQ_API_KEY is unset")
	}
}

func TestRequestAIReview_HappyPath(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-key")
	repo := newFakeRepo()
	id := repo.seedNote("author-1", "mis apuntes", StatusDraft)

	// Fake Groq server: assert the request shape and return a canned answer.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("expected bearer auth header, got %q", got)
		}
		var body openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("invalid request body: %v", err)
		}
		if len(body.Messages) != 2 || body.Messages[0].Role != "system" {
			t.Errorf("unexpected messages: %+v", body.Messages)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Buen resumen"}}]}`))
	}))
	defer srv.Close()

	svc := NewService(repo)
	svc.aiURL = srv.URL

	if err := svc.RequestAIReview(context.Background(), id, "mis apuntes"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastAIFeedback != "Buen resumen" {
		t.Errorf("expected feedback stored, got %q", repo.lastAIFeedback)
	}
	if repo.notes[id].Status != StatusAiReviewed {
		t.Errorf("expected status ai_reviewed, got %q", repo.notes[id].Status)
	}
	if repo.lastAILog == nil || !strings.Contains(repo.lastAILog.PromptUsed, "mis apuntes") {
		t.Errorf("expected the prompt to be logged, got %+v", repo.lastAILog)
	}
}

func TestRequestAIReview_Non200(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-key")
	repo := newFakeRepo()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	svc := NewService(repo)
	svc.aiURL = srv.URL

	if err := svc.RequestAIReview(context.Background(), "note-1", "x"); err == nil {
		t.Fatal("expected error when the AI returns a non-200 status")
	}
}

func TestRequestAIReview_EmptyChoices(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-key")
	repo := newFakeRepo()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	svc := NewService(repo)
	svc.aiURL = srv.URL

	if err := svc.RequestAIReview(context.Background(), "note-1", "x"); err == nil {
		t.Fatal("expected error when the AI returns no choices")
	}
}
