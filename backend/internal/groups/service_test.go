package groups

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// fakeRepo is an in-memory Repository so the service can be tested without a DB.
type fakeRepo struct {
	accounts map[string]*Account // userID -> account
	groups   map[string]*Group   // groupID -> group
	members  map[string][]Member // groupID -> roster
	tasks    map[string][]Task   // groupID -> tasks
	seq      int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		accounts: map[string]*Account{},
		groups:   map[string]*Group{},
		members:  map[string][]Member{},
		tasks:    map[string][]Task{},
	}
}

func (f *fakeRepo) id(prefix string) string {
	f.seq++
	return fmt.Sprintf("%s-%d", prefix, f.seq)
}

func (f *fakeRepo) AccountByID(_ context.Context, id string) (*Account, error) {
	acc, ok := f.accounts[id]
	if !ok {
		return nil, ErrForbidden
	}
	return acc, nil
}

func (f *fakeRepo) CreateGroup(_ context.Context, g *Group) error {
	g.ID = f.id("group")
	f.groups[g.ID] = g
	return nil
}

func (f *fakeRepo) GroupByID(_ context.Context, id string) (*Group, error) {
	g, ok := f.groups[id]
	if !ok {
		return nil, ErrGroupNotFound
	}
	return g, nil
}

func (f *fakeRepo) GroupsOwnedBy(_ context.Context, ownerID string) ([]Group, error) {
	var out []Group
	for _, g := range f.groups {
		if g.OwnerID == ownerID {
			out = append(out, *g)
		}
	}
	return out, nil
}

func (f *fakeRepo) GroupsForEmail(_ context.Context, email string) ([]Group, error) {
	var out []Group
	for groupID, roster := range f.members {
		for _, m := range roster {
			if m.Email == email {
				out = append(out, *f.groups[groupID])
			}
		}
	}
	return out, nil
}

func (f *fakeRepo) AddMembers(_ context.Context, groupID string, emails []string) error {
	for _, email := range emails {
		if f.hasMember(groupID, email) {
			continue
		}
		_, registered := f.accountByEmail(email)
		f.members[groupID] = append(f.members[groupID], Member{
			ID:         f.id("member"),
			Email:      email,
			Registered: registered,
		})
	}
	return nil
}

func (f *fakeRepo) ListMembers(_ context.Context, groupID string) ([]Member, error) {
	return f.members[groupID], nil
}

func (f *fakeRepo) RemoveMember(_ context.Context, groupID, memberID string) error {
	roster := f.members[groupID]
	for i, m := range roster {
		if m.ID == memberID {
			f.members[groupID] = append(roster[:i], roster[i+1:]...)
			return nil
		}
	}
	return ErrMemberNotFound
}

func (f *fakeRepo) IsMember(_ context.Context, groupID, email string) (bool, error) {
	return f.hasMember(groupID, email), nil
}

func (f *fakeRepo) CreateTask(_ context.Context, t *Task) error {
	t.ID = f.id("task")
	f.tasks[t.GroupID] = append(f.tasks[t.GroupID], *t)
	return nil
}

func (f *fakeRepo) ListTasks(_ context.Context, groupID string) ([]Task, error) {
	return f.tasks[groupID], nil
}

func (f *fakeRepo) hasMember(groupID, email string) bool {
	for _, m := range f.members[groupID] {
		if m.Email == email {
			return true
		}
	}
	return false
}

func (f *fakeRepo) accountByEmail(email string) (*Account, bool) {
	for _, acc := range f.accounts {
		if acc.Email == email {
			return acc, true
		}
	}
	return nil, false
}

// seedAccount registers an account and returns its user ID.
func (f *fakeRepo) seedAccount(role, email string) string {
	id := f.id("user")
	f.accounts[id] = &Account{ID: id, Role: role, Email: email}
	return id
}

func newServiceWith(repo *fakeRepo) *Service { return NewService(repo) }

func TestCreateGroup_TeacherOnly(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	teacher := repo.seedAccount(roleTeacher, "prof@unileon.es")
	student := repo.seedAccount("student", "alu@unileon.es")

	g, err := svc.CreateGroup(context.Background(), teacher, "Algebra")
	if err != nil {
		t.Fatalf("teacher should create a group: %v", err)
	}
	if g.OwnerID != teacher || g.Name != "Algebra" {
		t.Fatalf("unexpected group: %+v", g)
	}

	if _, err := svc.CreateGroup(context.Background(), student, "Algebra"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("student must not create groups, got %v", err)
	}
}

func TestCreateGroup_BlankNameRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	teacher := repo.seedAccount(roleTeacher, "prof@unileon.es")

	if _, err := svc.CreateGroup(context.Background(), teacher, "   "); !errors.Is(err, ErrValidation) {
		t.Fatalf("blank name must fail validation, got %v", err)
	}
}

func TestAddMembers_NormalizesAndDeduplicates(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	teacher := repo.seedAccount(roleTeacher, "prof@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), teacher, "Algebra")

	// Mixed case, surrounding spaces and a duplicate collapse to two entries.
	members, err := svc.AddMembers(context.Background(), teacher, g.ID, []string{
		"  Ada@Unileon.ES ", "ada@unileon.es", "bob@unileon.es", "",
	})
	if err != nil {
		t.Fatalf("AddMembers failed: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d (%+v)", len(members), members)
	}

	// Re-adding an existing email is a no-op.
	members, _ = svc.AddMembers(context.Background(), teacher, g.ID, []string{"ada@unileon.es"})
	if len(members) != 2 {
		t.Fatalf("re-adding should not duplicate, got %d", len(members))
	}
}

func TestAddMembers_InvalidEmailRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	teacher := repo.seedAccount(roleTeacher, "prof@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), teacher, "Algebra")

	if _, err := svc.AddMembers(context.Background(), teacher, g.ID, []string{"not-an-email"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("invalid email must fail validation, got %v", err)
	}
}

func TestAddMembers_NonOwnerForbidden(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	other := repo.seedAccount(roleTeacher, "otra@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")

	if _, err := svc.AddMembers(context.Background(), other, g.ID, []string{"alu@unileon.es"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-owner teacher must be forbidden, got %v", err)
	}
}

func TestListTasks_AccessControl(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	student := repo.seedAccount("student", "alu@unileon.es")
	outsider := repo.seedAccount("student", "ajeno@unileon.es")

	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	if _, err := svc.AddMembers(context.Background(), owner, g.ID, []string{"alu@unileon.es"}); err != nil {
		t.Fatalf("seed roster: %v", err)
	}
	if _, err := svc.CreateTask(context.Background(), owner, g.ID, TaskInput{Title: "Homework 1"}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if _, err := svc.ListTasks(context.Background(), owner, g.ID); err != nil {
		t.Fatalf("owner should see tasks: %v", err)
	}
	if _, err := svc.ListTasks(context.Background(), student, g.ID); err != nil {
		t.Fatalf("roster student should see tasks: %v", err)
	}
	if _, err := svc.ListTasks(context.Background(), outsider, g.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("outsider must be forbidden, got %v", err)
	}
}

func TestMyGroups_EmptyWhenNotInvited(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	student := repo.seedAccount("student", "alu@unileon.es")

	gs, err := svc.MyGroups(context.Background(), student)
	if err != nil {
		t.Fatalf("MyGroups failed: %v", err)
	}
	if len(gs) != 0 {
		t.Fatalf("a student with no invitations should have no groups, got %d", len(gs))
	}
}

func TestMyGroups_ResolvedByEmail(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	// Roster added BEFORE the student exists, then the student "registers".
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	if _, err := svc.AddMembers(context.Background(), owner, g.ID, []string{"late@unileon.es"}); err != nil {
		t.Fatalf("seed roster: %v", err)
	}
	student := repo.seedAccount("student", "late@unileon.es")

	gs, err := svc.MyGroups(context.Background(), student)
	if err != nil {
		t.Fatalf("MyGroups failed: %v", err)
	}
	if len(gs) != 1 || gs[0].ID != g.ID {
		t.Fatalf("student should auto-join by email match, got %+v", gs)
	}
}
