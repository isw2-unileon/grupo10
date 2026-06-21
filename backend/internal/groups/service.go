package groups

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxNameLength = 200
	roleTeacher   = "teacher"
	uploadDir     = "./uploads"
)

// Service implementa la capa de lógica de negocio para el LMS.
type Service struct {
	repo Repository
}

// NewService inicializa el servicio y asegura que la carpeta existe.
func NewService(repo Repository) *Service {
	_ = os.MkdirAll(uploadDir, 0750)
	return &Service{repo: repo}
}

// SaveUploadedFile copia un archivo físico al almacenamiento.
func (s *Service) SaveUploadedFile(fileReader io.Reader, filename string) (string, error) {
	cleanName := filepath.Base(filename)
	uniqueName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), cleanName)
	dstPath := filepath.Join(uploadDir, uniqueName)

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, fileReader); err != nil {
		return "", err
	}
	return uniqueName, nil
}

// CreateGroup crea un nuevo grupo.
func (s *Service) CreateGroup(ctx context.Context, userID, name string) (*Group, error) {
	acc, err := s.requireTeacher(ctx, userID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxNameLength {
		return nil, ErrValidation
	}
	g := &Group{Name: name, OwnerID: acc.ID}
	return g, s.repo.CreateGroup(ctx, g)
}

// GroupsOwned devuelve grupos creados por el docente.
func (s *Service) GroupsOwned(ctx context.Context, userID string) ([]Group, error) {
	acc, err := s.requireTeacher(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.GroupsOwnedBy(ctx, acc.ID)
}

// MyGroups devuelve los grupos donde el usuario es alumno.
func (s *Service) MyGroups(ctx context.Context, userID string) ([]Group, error) {
	acc, err := s.repo.AccountByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.GroupsForEmail(ctx, acc.Email)
}

// GroupDetail detalla información de un grupo.
func (s *Service) GroupDetail(ctx context.Context, userID, groupID string) (*Group, []Member, error) {
	g, err := s.ownedGroup(ctx, userID, groupID)
	if err != nil {
		return nil, nil, err
	}
	members, err := s.repo.ListMembers(ctx, g.ID)
	return g, members, err
}

// AddMembers matricula alumnos.
func (s *Service) AddMembers(ctx context.Context, userID, groupID string, emails []string) ([]Member, error) {
	g, err := s.ownedGroup(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	clean, err := sanitizeEmails(emails)
	if err != nil {
		return nil, err
	}
	if err := s.repo.AddMembers(ctx, g.ID, clean); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, g.ID)
}

// RemoveMember desmatricula a un estudiante.
func (s *Service) RemoveMember(ctx context.Context, userID, groupID, memberID string) error {
	g, err := s.ownedGroup(ctx, userID, groupID)
	if err != nil {
		return err
	}
	return s.repo.RemoveMember(ctx, g.ID, memberID)
}

// CreateSection crea un bloque o tema.
func (s *Service) CreateSection(ctx context.Context, userID, groupID string, title string, pos int) (*Section, error) {
	g, err := s.ownedGroup(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	sec := &Section{GroupID: g.ID, Title: title, Position: pos}
	return sec, s.repo.CreateSection(ctx, sec)
}

// UpdateSection edita el título de un tema.
func (s *Service) UpdateSection(ctx context.Context, userID, sectionID, title string) error {
	gID, err := s.repo.GetSectionGroup(ctx, sectionID)
	if err != nil {
		return err
	}
	if _, err := s.ownedGroup(ctx, userID, gID); err != nil {
		return err
	}
	return s.repo.UpdateSection(ctx, sectionID, title)
}

// DeleteSection borra un tema.
func (s *Service) DeleteSection(ctx context.Context, userID, sectionID string) error {
	gID, err := s.repo.GetSectionGroup(ctx, sectionID)
	if err != nil {
		return err
	}
	if _, err := s.ownedGroup(ctx, userID, gID); err != nil {
		return err
	}
	return s.repo.DeleteSection(ctx, sectionID)
}

// CreateResource crea material.
func (s *Service) CreateResource(ctx context.Context, userID, sectionID, rType, title, content, filePath string, dueAt *time.Time) (*Resource, error) {
	gID, err := s.repo.GetSectionGroup(ctx, sectionID)
	if err != nil {
		return nil, err
	}
	if _, err := s.ownedGroup(ctx, userID, gID); err != nil {
		return nil, err
	}

	res := &Resource{SectionID: sectionID, Type: rType, Title: title, Content: content, FilePath: filePath, DueAt: dueAt}
	return res, s.repo.CreateResource(ctx, res)
}

// CreateQuizWithQuestions crea un test.
func (s *Service) CreateQuizWithQuestions(ctx context.Context, userID, sectionID, title string, questions []QuizQuestion) (*Resource, error) {
	res, err := s.CreateResource(ctx, userID, sectionID, "quiz", title, "Cuestionario de evaluación", "", nil)
	if err != nil {
		return nil, err
	}

	for i, q := range questions {
		q.ResourceID = res.ID
		q.Position = i
		if err := s.repo.CreateQuizQuestion(ctx, &q); err != nil {
			return nil, err
		}
		for _, opt := range q.Options {
			opt.QuestionID = q.ID
			if err := s.repo.CreateQuizOption(ctx, &opt); err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}

// DeleteResource borra un material.
func (s *Service) DeleteResource(ctx context.Context, userID, resourceID string) error {
	res, err := s.repo.GetResourceByID(ctx, resourceID)
	if err != nil {
		return err
	}
	gID, err := s.repo.GetSectionGroup(ctx, res.SectionID)
	if err != nil {
		return err
	}
	if _, err := s.ownedGroup(ctx, userID, gID); err != nil {
		return err
	}
	return s.repo.DeleteResource(ctx, resourceID)
}

// GetGroupContent devuelve el árbol de contenidos completo.
func (s *Service) GetGroupContent(ctx context.Context, userID, groupID string) ([]Section, error) {
	acc, err := s.repo.AccountByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	g, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	if g.OwnerID != acc.ID {
		if isMem, _ := s.repo.IsMember(ctx, g.ID, acc.Email); !isMem {
			return nil, ErrForbidden
		}
	}

	sections, err := s.repo.GetSections(ctx, groupID)
	if err != nil {
		return nil, err
	}

	for i := range sections {
		resources, _ := s.repo.ListResourcesForSection(ctx, sections[i].ID)
		if acc.Role != roleTeacher {
			s.enrichResourcesForStudent(ctx, resources, acc.ID)
		}
		sections[i].Resources = resources
	}
	return sections, nil
}

// enrichResourcesForStudent inyecta metadatos a los recursos del alumno.
func (s *Service) enrichResourcesForStudent(ctx context.Context, resources []Resource, studentID string) {
	for j := range resources {
		hasSub, subAt, grade, _ := s.repo.HasSubmitted(ctx, resources[j].ID, studentID)
		resources[j].HasSubmitted = hasSub
		if hasSub {
			resources[j].SubmittedAt = &subAt
		}
		resources[j].CurrentGrade = grade

		if resources[j].Type == "assignment" && resources[j].DueAt != nil {
			compare := time.Now()
			if hasSub {
				compare = subAt
			}
			if compare.After(*resources[j].DueAt) {
				resources[j].IsLate = true
			}
		}
	}
}

// SubmitAssignment permite subir entregas.
func (s *Service) SubmitAssignment(ctx context.Context, userID, resourceID, textContent, filePath string) error {
	sub := &Submission{ResourceID: resourceID, StudentID: userID, TextContent: textContent, FilePath: filePath}
	return s.repo.SubmitAssignment(ctx, sub)
}

// GetAssignmentSubmissions recupera tareas.
func (s *Service) GetAssignmentSubmissions(ctx context.Context, userID, resourceID string) ([]Submission, error) {
	res, err := s.repo.GetResourceByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}
	gID, err := s.repo.GetSectionGroup(ctx, res.SectionID)
	if err != nil {
		return nil, err
	}
	if _, err := s.ownedGroup(ctx, userID, gID); err != nil {
		return nil, err
	}

	return s.repo.GetSubmissions(ctx, resourceID)
}

// GradeStudentTask califica a un estudiante.
func (s *Service) GradeStudentTask(ctx context.Context, userID, resourceID, studentID string, grade float64, feedback string) error {
	res, err := s.repo.GetResourceByID(ctx, resourceID)
	if err != nil {
		return err
	}
	gID, err := s.repo.GetSectionGroup(ctx, res.SectionID)
	if err != nil {
		return err
	}
	if _, err := s.ownedGroup(ctx, userID, gID); err != nil {
		return err
	}

	return s.repo.GradeSubmission(ctx, resourceID, studentID, grade, feedback)
}

// UpdateResource edita un recurso.
func (s *Service) UpdateResource(ctx context.Context, userID, resourceID, title, content string, dueAt *time.Time) error {
	res, err := s.repo.GetResourceByID(ctx, resourceID)
	if err != nil {
		return err
	}
	gID, err := s.repo.GetSectionGroup(ctx, res.SectionID)
	if err != nil {
		return err
	}
	if _, err := s.ownedGroup(ctx, userID, gID); err != nil {
		return err
	}

	res.Title = title
	res.Content = content
	res.DueAt = dueAt
	return s.repo.UpdateResource(ctx, res)
}

// GetQuiz recupera un test.
func (s *Service) GetQuiz(ctx context.Context, userID, resourceID string) (*Resource, error) {
	res, err := s.repo.GetResourceByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	qs, _ := s.repo.GetQuizQuestions(ctx, resourceID)
	acc, _ := s.repo.AccountByID(ctx, userID)

	for i, q := range qs {
		opts, _ := s.repo.GetQuizOptions(ctx, q.ID)
		if acc.Role != roleTeacher {
			for j := range opts {
				opts[j].IsCorrect = false
			}
		}
		qs[i].Options = opts
	}
	res.Questions = qs
	return res, nil
}

// SubmitQuiz evalúa el cuestionario.
func (s *Service) SubmitQuiz(ctx context.Context, userID, resourceID string, answers map[string]string) (float64, error) {
	qs, _ := s.repo.GetQuizQuestions(ctx, resourceID)
	if len(qs) == 0 {
		return 0, errors.New("vacío")
	}

	correctCount := 0
	for _, q := range qs {
		opts, _ := s.repo.GetQuizOptions(ctx, q.ID)
		selectedOpt := answers[q.ID]
		if selectedOpt != "" {
			_ = s.repo.SaveQuizAnswer(ctx, resourceID, userID, q.ID, selectedOpt)
		}

		for _, o := range opts {
			if o.ID == selectedOpt && o.IsCorrect {
				correctCount++
			}
		}
	}

	grade := (float64(correctCount) / float64(len(qs))) * 10.0
	sub := &Submission{
		ResourceID:  resourceID,
		StudentID:   userID,
		Grade:       &grade,
		Feedback:    "Auto-evaluado",
		TextContent: fmt.Sprintf("Aciertos: %d/%d", correctCount, len(qs)),
	}
	return grade, s.repo.SubmitAssignment(ctx, sub)
}

// GetQuizReview devuelve el test corregido.
func (s *Service) GetQuizReview(ctx context.Context, userID, resourceID, studentID string) (*Resource, error) {
	acc, err := s.repo.AccountByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if acc.Role != roleTeacher && userID != studentID {
		return nil, ErrForbidden
	}

	res, err := s.repo.GetResourceByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	qs, _ := s.repo.GetQuizQuestions(ctx, resourceID)
	studentAnswers, _ := s.repo.GetStudentAnswers(ctx, resourceID, studentID)

	for i, q := range qs {
		opts, _ := s.repo.GetQuizOptions(ctx, q.ID)
		selectedOpt := studentAnswers[q.ID]
		for j := range opts {
			if opts[j].ID == selectedOpt {
				opts[j].Selected = true
			}
		}
		qs[i].Options = opts
	}
	res.Questions = qs

	_, _, grade, _ := s.repo.HasSubmitted(ctx, resourceID, studentID)
	res.CurrentGrade = grade
	return res, nil
}

func (s *Service) requireTeacher(ctx context.Context, userID string) (*Account, error) {
	acc, err := s.repo.AccountByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if acc.Role != roleTeacher {
		return nil, ErrForbidden
	}
	return acc, nil
}

func (s *Service) ownedGroup(ctx context.Context, userID, groupID string) (*Group, error) {
	acc, err := s.repo.AccountByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	g, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if g.OwnerID != acc.ID {
		return nil, ErrForbidden
	}
	return g, nil
}

func sanitizeEmails(emails []string) ([]string, error) {
	seen := make(map[string]struct{})
	var clean []string
	for _, e := range emails {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if _, err := mail.ParseAddress(e); err != nil {
			return nil, ErrValidation
		}
		if _, dup := seen[e]; dup {
			continue
		}
		seen[e] = struct{}{}
		clean = append(clean, e)
	}
	return clean, nil
}
