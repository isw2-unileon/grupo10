//go:build integration

// Integration tests for the Postgres repository. They run against a real
// Postgres instance and are gated behind the `integration` build tag so the
// default `go test` (unit) run stays DB-free.
//
//	# locally
//	docker run --rm -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:17-alpine
//	TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" \
//	  go test -tags integration ./backend/internal/notes/
package notes

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

var testDB *sql.DB

// TestMain opens the DB once and applies the full schema before running the
// integration tests. If TEST_DATABASE_URL is unset, every test is skipped.
func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		os.Exit(0) // nothing to do without a database
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic("could not open DB: " + err.Error())
	}
	if err := db.Ping(); err != nil {
		panic("could not reach DB: " + err.Error())
	}

	schema, err := os.ReadFile("../../migrations/up.sql")
	if err != nil {
		panic("could not read up.sql: " + err.Error())
	}
	if _, err := db.Exec(string(schema)); err != nil {
		panic("could not apply migrations: " + err.Error())
	}

	testDB = db
	os.Exit(m.Run())
}

// newRepo returns a repository backed by the real DB, after wiping the data
// tables so each test starts from a clean, isolated state. Roles survive the
// truncation because the schema seeds them and notes depend on them.
func newRepo(t *testing.T) *PostgresRepository {
	t.Helper()
	if testDB == nil {
		t.Skip("TEST_DATABASE_URL not set; skipping Postgres integration tests")
	}
	_, err := testDB.Exec(`TRUNCATE users, notes, note_shares, ai_feedback_logs,
		class_groups, group_members RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("could not reset tables: %v", err)
	}
	return NewPostgresRepository(testDB)
}

// seedUser inserts a user with the given email and returns its generated ID.
func seedUser(t *testing.T, email string) string {
	t.Helper()
	var id string
	err := testDB.QueryRow(`
		INSERT INTO users (role_id, name, email, password_hash)
		VALUES ((SELECT id FROM roles WHERE name = 'student'), 'Test', $1, 'x')
		RETURNING id`, email).Scan(&id)
	if err != nil {
		t.Fatalf("could not seed user: %v", err)
	}
	return id
}

func TestIntegration_CreateAndGetByID(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	author := seedUser(t, "author@uni.es")

	n := &Note{AuthorID: author, Title: "T", Content: "C", Status: StatusDraft}
	if err := repo.CreateNote(ctx, n); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if n.ID == "" || n.CreatedAt.IsZero() {
		t.Fatalf("expected ID and timestamps populated, got %+v", n)
	}

	got, err := repo.GetByID(ctx, n.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != "T" || got.AuthorID != author || got.Status != StatusDraft {
		t.Errorf("unexpected note: %+v", got)
	}
}

func TestIntegration_GetByID_NotFound(t *testing.T) {
	repo := newRepo(t)

	_, err := repo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("expected ErrNoteNotFound, got %v", err)
	}
}

func TestIntegration_GetByAuthor(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	me := seedUser(t, "me@uni.es")
	other := seedUser(t, "other@uni.es")
	mustCreate(t, repo, me, "mine")
	mustCreate(t, repo, other, "theirs")

	list, err := repo.GetByAuthor(ctx, me)
	if err != nil {
		t.Fatalf("GetByAuthor: %v", err)
	}
	if len(list) != 1 || list[0].Title != "mine" {
		t.Errorf("expected only the author's note, got %+v", list)
	}
}

func TestIntegration_UpdateNote_OwnershipEnforced(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	owner := seedUser(t, "owner@uni.es")
	id := mustCreate(t, repo, owner, "old")

	// Wrong author: nothing updated -> ErrNoteNotFound.
	err := repo.UpdateNote(ctx, &Note{ID: id, AuthorID: "00000000-0000-0000-0000-000000000000", Title: "x", Content: "x"})
	if !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("expected ErrNoteNotFound for non-owner, got %v", err)
	}

	if err := repo.UpdateNote(ctx, &Note{ID: id, AuthorID: owner, Title: "new", Content: "updated"}); err != nil {
		t.Fatalf("UpdateNote: %v", err)
	}
	got, _ := repo.GetByID(ctx, id)
	if got.Content != "updated" {
		t.Errorf("note not updated: %+v", got)
	}
}

func TestIntegration_StatusPipeline(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	author := seedUser(t, "author@uni.es")
	id := mustCreate(t, repo, author, "note")

	if err := repo.UpdateStatus(ctx, id, StatusPending); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	pending, err := repo.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending: %v", err)
	}
	if len(pending) != 1 || pending[0].ID != id {
		t.Errorf("expected the pending note, got %+v", pending)
	}

	if err := repo.ApproveNoteWithFeedback(ctx, id, "well done"); err != nil {
		t.Fatalf("ApproveNoteWithFeedback: %v", err)
	}
	list, _ := repo.GetByAuthor(ctx, author)
	if list[0].Status != StatusApproved || list[0].TeacherFeedback == nil || *list[0].TeacherFeedback != "well done" {
		t.Errorf("note not approved with feedback: %+v", list[0])
	}
}

func TestIntegration_UpdateNoteWithAI_WritesLog(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	author := seedUser(t, "author@uni.es")
	id := mustCreate(t, repo, author, "note")

	log := &AIFeedbackLog{NoteID: id, PromptUsed: "prompt", Response: "feedback"}
	if err := repo.UpdateNoteWithAI(ctx, id, "feedback", log); err != nil {
		t.Fatalf("UpdateNoteWithAI: %v", err)
	}

	got, _ := repo.GetByID(ctx, id)
	if got.Status != StatusAiReviewed || got.AiFeedback == nil || *got.AiFeedback != "feedback" {
		t.Errorf("AI feedback not stored: %+v", got)
	}

	var logCount int
	if err := testDB.QueryRow(`SELECT COUNT(*) FROM ai_feedback_logs WHERE note_id = $1`, id).Scan(&logCount); err != nil {
		t.Fatalf("count logs: %v", err)
	}
	if logCount != 1 {
		t.Errorf("expected 1 ai_feedback_log row, got %d", logCount)
	}
}

func TestIntegration_DeleteNote(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	author := seedUser(t, "author@uni.es")
	id := mustCreate(t, repo, author, "note")

	if err := repo.DeleteNote(ctx, id, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("expected ErrNoteNotFound deleting as non-owner, got %v", err)
	}
	if err := repo.DeleteNote(ctx, id, author); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}
	if _, err := repo.GetByID(ctx, id); !errors.Is(err, ErrNoteNotFound) {
		t.Errorf("note still present after delete")
	}
}

func TestIntegration_GetUserEmail(t *testing.T) {
	repo := newRepo(t)
	id := seedUser(t, "lookup@uni.es")

	email, err := repo.GetUserEmail(context.Background(), id)
	if err != nil {
		t.Fatalf("GetUserEmail: %v", err)
	}
	if email != "lookup@uni.es" {
		t.Errorf("expected lookup@uni.es, got %q", email)
	}
}

func TestIntegration_ShareByEmail(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	author := seedUser(t, "author@uni.es")
	id := mustCreate(t, repo, author, "shared note")

	friend := "friend@uni.es"
	if err := repo.ShareNote(ctx, id, &friend, nil); err != nil {
		t.Fatalf("ShareNote: %v", err)
	}

	list, err := repo.GetSharedWithMe(ctx, friend)
	if err != nil {
		t.Fatalf("GetSharedWithMe: %v", err)
	}
	if len(list) != 1 || list[0].ID != id {
		t.Errorf("expected the shared note, got %+v", list)
	}
}

func TestIntegration_ShareByGroup(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()
	author := seedUser(t, "author@uni.es")
	id := mustCreate(t, repo, author, "group note")

	// A class group whose roster contains a member email.
	var groupID string
	if err := testDB.QueryRow(`
		INSERT INTO class_groups (name, owner_id) VALUES ('Class', $1) RETURNING id`, author).Scan(&groupID); err != nil {
		t.Fatalf("seed group: %v", err)
	}
	member := "member@uni.es"
	if _, err := testDB.Exec(`INSERT INTO group_members (group_id, email) VALUES ($1, $2)`, groupID, member); err != nil {
		t.Fatalf("seed group member: %v", err)
	}

	if err := repo.ShareNote(ctx, id, nil, &groupID); err != nil {
		t.Fatalf("ShareNote (group): %v", err)
	}

	list, err := repo.GetSharedWithMe(ctx, member)
	if err != nil {
		t.Fatalf("GetSharedWithMe: %v", err)
	}
	if len(list) != 1 || list[0].ID != id {
		t.Errorf("expected the group-shared note, got %+v", list)
	}
}

// mustCreate inserts a draft note for author and returns its ID.
func mustCreate(t *testing.T, repo *PostgresRepository, author, title string) string {
	t.Helper()
	n := &Note{AuthorID: author, Title: title, Content: "body", Status: StatusDraft}
	if err := repo.CreateNote(context.Background(), n); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	return n.ID
}
