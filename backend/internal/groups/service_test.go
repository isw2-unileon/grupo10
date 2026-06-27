package groups

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// fakeRepo is an in-memory Repository so the service can be tested without a DB.
type fakeRepo struct {
	accounts  map[string]*Account  // userID -> account
	groups    map[string]*Group    // groupID -> group
	members   map[string][]Member  // groupID -> roster
	sections  map[string]*Section  // sectionID -> section
	resources map[string]*Resource // resourceID -> resource

	questions   map[string]*QuizQuestion // questionID -> question
	options     map[string][]QuizOption  // questionID -> options
	answers     map[string]string        // resourceID|studentID|questionID -> optionID
	submissions map[string][]Submission  // resourceID -> submissions
	analytics   map[string][]SubjectStat // studentID -> analytics

	deletedSections  map[string]bool // sectionID -> deleted
	deletedResources map[string]bool // resourceID -> deleted

	seq int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		accounts:  map[string]*Account{},
		groups:    map[string]*Group{},
		members:   map[string][]Member{},
		sections:  map[string]*Section{},
		resources: map[string]*Resource{},

		questions:   map[string]*QuizQuestion{},
		options:     map[string][]QuizOption{},
		answers:     map[string]string{},
		submissions: map[string][]Submission{},
		analytics:   map[string][]SubjectStat{},

		deletedSections:  map[string]bool{},
		deletedResources: map[string]bool{},
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

// ==========================================
// NUEVOS MÉTODOS MOODLE PARA PASAR EL LINTER
// ==========================================

func (f *fakeRepo) CreateSection(_ context.Context, sec *Section) error {
	sec.ID = f.id("section")
	f.sections[sec.ID] = sec
	return nil
}

func (f *fakeRepo) UpdateSection(_ context.Context, sectionID, title string) error {
	if s, ok := f.sections[sectionID]; ok {
		s.Title = title
	}
	return nil
}

func (f *fakeRepo) DeleteSection(_ context.Context, sectionID string) error {
	f.deletedSections[sectionID] = true
	delete(f.sections, sectionID)
	return nil
}

func (f *fakeRepo) GetSections(_ context.Context, groupID string) ([]Section, error) {
	var out []Section
	for _, s := range f.sections {
		if s.GroupID == groupID {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetSectionGroup(_ context.Context, sectionID string) (string, error) {
	s, ok := f.sections[sectionID]
	if !ok {
		return "", errors.New("section not found")
	}
	return s.GroupID, nil
}

func (f *fakeRepo) CreateResource(_ context.Context, res *Resource) error {
	res.ID = f.id("resource")
	f.resources[res.ID] = res
	return nil
}

func (f *fakeRepo) UpdateResource(_ context.Context, res *Resource) error {
	if _, ok := f.resources[res.ID]; ok {
		f.resources[res.ID] = res
	}
	return nil
}

func (f *fakeRepo) DeleteResource(_ context.Context, resourceID string) error {
	f.deletedResources[resourceID] = true
	delete(f.resources, resourceID)
	return nil
}
func (f *fakeRepo) GetResourceByID(_ context.Context, resourceID string) (*Resource, error) {
	r, ok := f.resources[resourceID]
	if !ok {
		return nil, errors.New("not found")
	}
	return r, nil
}

func (f *fakeRepo) ListResourcesForSection(_ context.Context, sectionID string) ([]Resource, error) {
	var out []Resource
	for _, r := range f.resources {
		if r.SectionID == sectionID {
			out = append(out, *r)
		}
	}
	return out, nil
}

// In-memory implementations of the quiz / submission / analytics surface so the
// service's scoring and grading logic can be exercised without a database.

func (f *fakeRepo) CreateQuizQuestion(_ context.Context, q *QuizQuestion) error {
	q.ID = f.id("question")
	stored := *q
	f.questions[q.ID] = &stored
	return nil
}

func (f *fakeRepo) CreateQuizOption(_ context.Context, opt *QuizOption) error {
	opt.ID = f.id("option")
	f.options[opt.QuestionID] = append(f.options[opt.QuestionID], *opt)
	return nil
}

func (f *fakeRepo) GetQuizQuestions(_ context.Context, resourceID string) ([]QuizQuestion, error) {
	var out []QuizQuestion
	for _, q := range f.questions {
		if q.ResourceID == resourceID {
			out = append(out, *q)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out, nil
}

func (f *fakeRepo) GetQuizOptions(_ context.Context, questionID string) ([]QuizOption, error) {
	return f.options[questionID], nil
}

func answerKey(resourceID, studentID, questionID string) string {
	return resourceID + "|" + studentID + "|" + questionID
}

func (f *fakeRepo) SaveQuizAnswer(_ context.Context, resourceID, studentID, questionID, optionID string) error {
	f.answers[answerKey(resourceID, studentID, questionID)] = optionID
	return nil
}

func (f *fakeRepo) GetStudentAnswers(_ context.Context, resourceID, studentID string) (map[string]string, error) {
	out := map[string]string{}
	prefix := resourceID + "|" + studentID + "|"
	for k, optID := range f.answers {
		if strings.HasPrefix(k, prefix) {
			out[strings.TrimPrefix(k, prefix)] = optID
		}
	}
	return out, nil
}

func (f *fakeRepo) SubmitAssignment(_ context.Context, sub *Submission) error {
	sub.ID = f.id("submission")
	sub.SubmittedAt = time.Now()
	f.submissions[sub.ResourceID] = append(f.submissions[sub.ResourceID], *sub)
	return nil
}

func (f *fakeRepo) GradeSubmission(_ context.Context, resourceID, studentID string, grade float64, feedback string) error {
	subs := f.submissions[resourceID]
	for i := range subs {
		if subs[i].StudentID == studentID {
			g := grade
			subs[i].Grade = &g
			subs[i].Feedback = feedback
		}
	}
	return nil
}

func (f *fakeRepo) GetSubmissions(_ context.Context, resourceID string) ([]Submission, error) {
	return f.submissions[resourceID], nil
}

func (f *fakeRepo) HasSubmitted(_ context.Context, resourceID, studentID string) (bool, time.Time, *float64, error) {
	subs := f.submissions[resourceID]
	for i := len(subs) - 1; i >= 0; i-- {
		if subs[i].StudentID == studentID {
			return true, subs[i].SubmittedAt, subs[i].Grade, nil
		}
	}
	return false, time.Time{}, nil, nil
}

func (f *fakeRepo) GetStudentAnalytics(_ context.Context, studentID string) ([]SubjectStat, error) {
	return f.analytics[studentID], nil
}

// ==========================================

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

// ADAPTADO: Ahora comprueba el acceso a GetGroupContent en lugar de ListTasks
func TestGetGroupContent_AccessControl(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	student := repo.seedAccount("student", "alu@unileon.es")
	outsider := repo.seedAccount("student", "ajeno@unileon.es")

	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	if _, err := svc.AddMembers(context.Background(), owner, g.ID, []string{"alu@unileon.es"}); err != nil {
		t.Fatalf("seed roster: %v", err)
	}

	// El docente (professor) crea una sección (sustituye a la antigua tarea)
	if _, err := svc.CreateSection(context.Background(), owner, g.ID, "Tema 1", 0); err != nil {
		t.Fatalf("create section: %v", err)
	}

	if _, err := svc.GetGroupContent(context.Background(), owner, g.ID); err != nil {
		t.Fatalf("owner should see content: %v", err)
	}
	if _, err := svc.GetGroupContent(context.Background(), student, g.ID); err != nil {
		t.Fatalf("roster student should see content: %v", err)
	}
	if _, err := svc.GetGroupContent(context.Background(), outsider, g.ID); !errors.Is(err, ErrForbidden) {
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
