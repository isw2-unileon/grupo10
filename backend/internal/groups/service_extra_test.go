package groups

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Groups: detail, ownership listing and member removal
// ---------------------------------------------------------------------------

func TestGroupDetail_OwnerSeesRosterOutsiderForbidden(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	other := repo.seedAccount(roleTeacher, "otra@unileon.es")

	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	if _, err := svc.AddMembers(context.Background(), owner, g.ID, []string{"alu@unileon.es"}); err != nil {
		t.Fatalf("seed roster: %v", err)
	}

	gotGroup, members, err := svc.GroupDetail(context.Background(), owner, g.ID)
	if err != nil {
		t.Fatalf("owner should read detail: %v", err)
	}
	if gotGroup.ID != g.ID || len(members) != 1 {
		t.Fatalf("unexpected detail: group=%+v members=%d", gotGroup, len(members))
	}

	if _, _, err := svc.GroupDetail(context.Background(), other, g.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-owner must be forbidden, got %v", err)
	}
}

func TestGroupsOwned_TeacherOnly(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	teacher := repo.seedAccount(roleTeacher, "prof@unileon.es")
	student := repo.seedAccount("student", "alu@unileon.es")

	if _, err := svc.CreateGroup(context.Background(), teacher, "Algebra"); err != nil {
		t.Fatalf("create group: %v", err)
	}

	owned, err := svc.GroupsOwned(context.Background(), teacher)
	if err != nil || len(owned) != 1 {
		t.Fatalf("teacher should own 1 group, got %d (%v)", len(owned), err)
	}

	if _, err := svc.GroupsOwned(context.Background(), student); !errors.Is(err, ErrForbidden) {
		t.Fatalf("student must not list owned groups, got %v", err)
	}
}

func TestRemoveMember_RemovesFromRoster(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	members, _ := svc.AddMembers(context.Background(), owner, g.ID, []string{"alu@unileon.es"})

	if err := svc.RemoveMember(context.Background(), owner, g.ID, members[0].ID); err != nil {
		t.Fatalf("owner should remove a member: %v", err)
	}

	left, _ := repo.ListMembers(context.Background(), g.ID)
	if len(left) != 0 {
		t.Fatalf("roster should be empty after removal, got %d", len(left))
	}
}

// ---------------------------------------------------------------------------
// Sections and resources: ownership is enforced through the parent group
// ---------------------------------------------------------------------------

func TestSection_UpdateAndDelete_OwnershipEnforced(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	other := repo.seedAccount(roleTeacher, "otra@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	sec, _ := svc.CreateSection(context.Background(), owner, g.ID, "Tema 1", 0)

	// A teacher who does not own the group cannot touch its sections.
	if err := svc.UpdateSection(context.Background(), other, sec.ID, "Hack"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-owner update must be forbidden, got %v", err)
	}

	if err := svc.UpdateSection(context.Background(), owner, sec.ID, "Tema 1 (revisado)"); err != nil {
		t.Fatalf("owner should update section: %v", err)
	}
	if repo.sections[sec.ID].Title != "Tema 1 (revisado)" {
		t.Fatalf("title was not updated: %q", repo.sections[sec.ID].Title)
	}

	if err := svc.DeleteSection(context.Background(), owner, sec.ID); err != nil {
		t.Fatalf("owner should delete section: %v", err)
	}
	if !repo.deletedSections[sec.ID] {
		t.Fatalf("section was not deleted")
	}
}

func TestResource_CreateUpdateDelete(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	sec, _ := svc.CreateSection(context.Background(), owner, g.ID, "Tema 1", 0)

	res, err := svc.CreateResource(context.Background(), owner, sec.ID, "file", "Apuntes", "contenido", "apuntes.pdf", nil)
	if err != nil {
		t.Fatalf("create resource: %v", err)
	}

	due := time.Now().Add(48 * time.Hour)
	if err := svc.UpdateResource(context.Background(), owner, res.ID, "Apuntes v2", "nuevo", &due); err != nil {
		t.Fatalf("update resource: %v", err)
	}
	stored := repo.resources[res.ID]
	if stored.Title != "Apuntes v2" || stored.Content != "nuevo" || stored.DueAt == nil {
		t.Fatalf("resource not updated: %+v", stored)
	}

	if err := svc.DeleteResource(context.Background(), owner, res.ID); err != nil {
		t.Fatalf("delete resource: %v", err)
	}
	if !repo.deletedResources[res.ID] {
		t.Fatalf("resource was not deleted")
	}
}

func TestGetGroupContent_FlagsLateAssignmentForStudent(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	student := repo.seedAccount("student", "alu@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	if _, err := svc.AddMembers(context.Background(), owner, g.ID, []string{"alu@unileon.es"}); err != nil {
		t.Fatalf("seed roster: %v", err)
	}
	sec, _ := svc.CreateSection(context.Background(), owner, g.ID, "Tema 1", 0)

	past := time.Now().Add(-1 * time.Hour)
	if _, err := svc.CreateResource(context.Background(), owner, sec.ID, "assignment", "Entrega", "", "", &past); err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	sections, err := svc.GetGroupContent(context.Background(), student, g.ID)
	if err != nil {
		t.Fatalf("student should read content: %v", err)
	}
	if len(sections) != 1 || len(sections[0].Resources) != 1 {
		t.Fatalf("expected one resource, got %+v", sections)
	}
	r := sections[0].Resources[0]
	if !r.IsLate || r.HasSubmitted {
		t.Fatalf("overdue unsubmitted assignment should be late and not submitted: %+v", r)
	}
}

// ---------------------------------------------------------------------------
// Quizzes: creation, student visibility, scoring and review
// ---------------------------------------------------------------------------

// seedQuiz creates a two-question quiz and returns the resource together with
// the question and correct-option IDs (read back as the teacher, who sees the
// answer key).
func seedQuiz(t *testing.T, svc *Service, owner, sectionID string) (res *Resource, correct map[string]string) {
	t.Helper()
	questions := []QuizQuestion{
		{QuestionText: "2+2?", Options: []QuizOption{{OptionText: "4", IsCorrect: true}, {OptionText: "5"}}},
		{QuestionText: "Capital de España?", Options: []QuizOption{{OptionText: "Madrid", IsCorrect: true}, {OptionText: "París"}}},
	}
	res, err := svc.CreateQuizWithQuestions(context.Background(), owner, sectionID, "Test 1", questions)
	if err != nil {
		t.Fatalf("create quiz: %v", err)
	}

	teacherView, err := svc.GetQuiz(context.Background(), owner, res.ID)
	if err != nil {
		t.Fatalf("teacher GetQuiz: %v", err)
	}
	if len(teacherView.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(teacherView.Questions))
	}

	correct = map[string]string{}
	for _, q := range teacherView.Questions {
		for _, o := range q.Options {
			if o.IsCorrect {
				correct[q.ID] = o.ID
			}
		}
	}
	if len(correct) != 2 {
		t.Fatalf("expected a correct option per question, got %d", len(correct))
	}
	return res, correct
}

func TestGetQuiz_HidesAnswerKeyFromStudents(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	student := repo.seedAccount("student", "alu@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	sec, _ := svc.CreateSection(context.Background(), owner, g.ID, "Tema 1", 0)
	res, _ := seedQuiz(t, svc, owner, sec.ID)

	studentView, err := svc.GetQuiz(context.Background(), student, res.ID)
	if err != nil {
		t.Fatalf("student GetQuiz: %v", err)
	}
	for _, q := range studentView.Questions {
		for _, o := range q.Options {
			if o.IsCorrect {
				t.Fatalf("student must not see which option is correct: %+v", o)
			}
		}
	}
}

func TestSubmitQuiz_ScoresAllCorrectAndPartial(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	student := repo.seedAccount("student", "alu@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	sec, _ := svc.CreateSection(context.Background(), owner, g.ID, "Tema 1", 0)
	res, correct := seedQuiz(t, svc, owner, sec.ID)

	// All correct -> 10.
	grade, err := svc.SubmitQuiz(context.Background(), student, res.ID, correct)
	if err != nil {
		t.Fatalf("submit quiz: %v", err)
	}
	if grade != 10 {
		t.Fatalf("all-correct quiz should score 10, got %v", grade)
	}

	// The submission is recorded so HasSubmitted reflects it.
	done, _, storedGrade, _ := repo.HasSubmitted(context.Background(), res.ID, student)
	if !done || storedGrade == nil || *storedGrade != 10 {
		t.Fatalf("submission/grade not persisted: done=%v grade=%v", done, storedGrade)
	}

	// Answer only the first question correctly -> 5 out of 10.
	partial := map[string]string{}
	first := true
	for qID, optID := range correct {
		if first {
			partial[qID] = optID // correct
			first = false
		} else {
			partial[qID] = "wrong-option"
		}
	}
	grade, err = svc.SubmitQuiz(context.Background(), student, res.ID, partial)
	if err != nil {
		t.Fatalf("submit partial quiz: %v", err)
	}
	if grade != 5 {
		t.Fatalf("half-correct quiz should score 5, got %v", grade)
	}
}

func TestSubmitQuiz_EmptyQuizErrors(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	student := repo.seedAccount("student", "alu@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	sec, _ := svc.CreateSection(context.Background(), owner, g.ID, "Tema 1", 0)

	// A quiz resource with no questions.
	res, err := svc.CreateQuizWithQuestions(context.Background(), owner, sec.ID, "Vacío", nil)
	if err != nil {
		t.Fatalf("create empty quiz: %v", err)
	}

	if _, err := svc.SubmitQuiz(context.Background(), student, res.ID, map[string]string{}); err == nil {
		t.Fatalf("submitting an empty quiz must fail")
	}
}

func TestGetQuizReview_MarksSelectedAndGuardsAccess(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	student := repo.seedAccount("student", "alu@unileon.es")
	other := repo.seedAccount("student", "otro@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	sec, _ := svc.CreateSection(context.Background(), owner, g.ID, "Tema 1", 0)
	res, correct := seedQuiz(t, svc, owner, sec.ID)

	if _, err := svc.SubmitQuiz(context.Background(), student, res.ID, correct); err != nil {
		t.Fatalf("submit quiz: %v", err)
	}

	// The owner reviews the student's attempt and sees the selected options.
	review, err := svc.GetQuizReview(context.Background(), owner, res.ID, student)
	if err != nil {
		t.Fatalf("owner review: %v", err)
	}
	selected := 0
	for _, q := range review.Questions {
		for _, o := range q.Options {
			if o.Selected {
				selected++
			}
		}
	}
	if selected != 2 {
		t.Fatalf("expected 2 selected options across the review, got %d", selected)
	}

	// A different student cannot peek at someone else's review.
	if _, err := svc.GetQuizReview(context.Background(), other, res.ID, student); !errors.Is(err, ErrForbidden) {
		t.Fatalf("foreign student review must be forbidden, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Assignments: submit, list and grade
// ---------------------------------------------------------------------------

func TestAssignment_SubmitListAndGrade(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	student := repo.seedAccount("student", "alu@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	sec, _ := svc.CreateSection(context.Background(), owner, g.ID, "Tema 1", 0)
	res, _ := svc.CreateResource(context.Background(), owner, sec.ID, "assignment", "Entrega", "", "", nil)

	if err := svc.SubmitAssignment(context.Background(), student, res.ID, "mi respuesta", ""); err != nil {
		t.Fatalf("submit assignment: %v", err)
	}

	subs, err := svc.GetAssignmentSubmissions(context.Background(), owner, res.ID)
	if err != nil || len(subs) != 1 {
		t.Fatalf("teacher should list 1 submission, got %d (%v)", len(subs), err)
	}

	if err := svc.GradeStudentTask(context.Background(), owner, res.ID, student, 8.5, "buen trabajo"); err != nil {
		t.Fatalf("grade task: %v", err)
	}
	graded, _ := svc.GetAssignmentSubmissions(context.Background(), owner, res.ID)
	if graded[0].Grade == nil || *graded[0].Grade != 8.5 || graded[0].Feedback != "buen trabajo" {
		t.Fatalf("grade/feedback not stored: %+v", graded[0])
	}
}

func TestGetAssignmentSubmissions_NonOwnerForbidden(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	other := repo.seedAccount(roleTeacher, "otra@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	sec, _ := svc.CreateSection(context.Background(), owner, g.ID, "Tema 1", 0)
	res, _ := svc.CreateResource(context.Background(), owner, sec.ID, "assignment", "Entrega", "", "", nil)

	if _, err := svc.GetAssignmentSubmissions(context.Background(), other, res.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-owner must not read submissions, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Analytics: student profile and teacher-facing stats
// ---------------------------------------------------------------------------

func TestGetStudentProfile_ReturnsAnalytics(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	student := repo.seedAccount("student", "alu@unileon.es")
	repo.analytics[student] = []SubjectStat{{GroupID: "g1", GroupName: "Algebra", TotalAverage: 7.5}}

	profile, err := svc.GetStudentProfile(context.Background(), student)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.Email != "alu@unileon.es" || profile.Role != "student" {
		t.Fatalf("unexpected identity: %+v", profile)
	}
	if len(profile.Analytics) != 1 || profile.Analytics[0].TotalAverage != 7.5 {
		t.Fatalf("analytics not surfaced: %+v", profile.Analytics)
	}
}

func TestGetStudentStatsForTeacher_ReturnsRecordedStat(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	student := repo.seedAccount("student", "alu@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	repo.analytics[student] = []SubjectStat{{GroupID: g.ID, GroupName: "Algebra", TotalAverage: 9}}

	stat, err := svc.GetStudentStatsForTeacher(context.Background(), owner, g.ID, student)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stat.TotalAverage != 9 {
		t.Fatalf("expected recorded stat, got %+v", stat)
	}
}

func TestGetStudentStatsForTeacher_FallsBackToSectionsWithoutData(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	student := repo.seedAccount("student", "alu@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")
	if _, err := svc.CreateSection(context.Background(), owner, g.ID, "Tema 1", 0); err != nil {
		t.Fatalf("create section: %v", err)
	}

	// The student has no analytics yet: the teacher still gets the real sections
	// at zero so the UI does not show "no structured topics".
	stat, err := svc.GetStudentStatsForTeacher(context.Background(), owner, g.ID, student)
	if err != nil {
		t.Fatalf("stats fallback: %v", err)
	}
	if stat.TotalAverage != 0 || len(stat.Sections) != 1 || stat.Sections[0].Average != 0 {
		t.Fatalf("expected zeroed sections fallback, got %+v", stat)
	}
}

func TestGetStudentStatsForTeacher_NonOwnerForbidden(t *testing.T) {
	repo := newFakeRepo()
	svc := newServiceWith(repo)
	owner := repo.seedAccount(roleTeacher, "prof@unileon.es")
	other := repo.seedAccount(roleTeacher, "otra@unileon.es")
	student := repo.seedAccount("student", "alu@unileon.es")
	g, _ := svc.CreateGroup(context.Background(), owner, "Algebra")

	if _, err := svc.GetStudentStatsForTeacher(context.Background(), other, g.ID, student); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-owner teacher must be forbidden, got %v", err)
	}
}
