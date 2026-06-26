package groups

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/isw2-unileon/grupo10/backend/internal/notes"
	"github.com/isw2-unileon/grupo10/backend/internal/users"
)

// Handler expone las rutas HTTP para la gestión de grupos.
type Handler struct {
	svc    *Service
	parser users.TokenParser
}

// NewHandler inicializa un nuevo Handler.
func NewHandler(svc *Service, parser users.TokenParser) *Handler {
	return &Handler{svc: svc, parser: parser}
}

// RegisterRoutes registra todas las rutas REST.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	auth := users.RequireAuth(h.parser)

	mux.Handle("POST /api/groups", auth(http.HandlerFunc(h.createGroup)))
	mux.Handle("GET /api/groups", auth(http.HandlerFunc(h.listOwnedGroups)))
	mux.Handle("GET /api/me/groups", auth(http.HandlerFunc(h.listMyGroups)))
	mux.Handle("GET /api/groups/{id}", auth(http.HandlerFunc(h.groupDetail)))

	mux.Handle("POST /api/groups/{id}/members", auth(http.HandlerFunc(h.addMembers)))
	mux.Handle("DELETE /api/groups/{id}/members/{memberID}", auth(http.HandlerFunc(h.removeMember)))

	mux.Handle("POST /api/groups/{id}/sections", auth(http.HandlerFunc(h.createSection)))
	mux.Handle("PUT /api/sections/{sectionId}", auth(http.HandlerFunc(h.updateSection)))
	mux.Handle("DELETE /api/sections/{sectionId}", auth(http.HandlerFunc(h.deleteSection)))

	mux.Handle("POST /api/sections/{sectionId}/resources", auth(http.HandlerFunc(h.createResource)))
	mux.Handle("POST /api/sections/{sectionId}/quizzes", auth(http.HandlerFunc(h.createQuiz)))
	mux.Handle("POST /api/quizzes/ai-generate", auth(http.HandlerFunc(h.aiGenerateQuiz)))
	mux.Handle("POST /api/quizzes/ai-improve", auth(http.HandlerFunc(h.aiImproveQuiz)))
	mux.Handle("DELETE /api/resources/{resourceId}", auth(http.HandlerFunc(h.deleteResource)))
	mux.Handle("GET /api/groups/{id}/content", auth(http.HandlerFunc(h.getGroupContent)))

	mux.Handle("POST /api/resources/{resourceId}/submit", auth(http.HandlerFunc(h.submitAssignment)))
	mux.Handle("GET /api/resources/{resourceId}/submissions", auth(http.HandlerFunc(h.listSubmissions)))
	mux.Handle("POST /api/resources/{resourceId}/submissions/{studentId}/grade", auth(http.HandlerFunc(h.gradeSubmission)))

	mux.Handle("GET /api/uploads/{filename}", auth(http.HandlerFunc(h.serveFile)))
	mux.Handle("PUT /api/resources/{resourceId}", auth(http.HandlerFunc(h.updateResource)))
	mux.Handle("GET /api/resources/{resourceId}/quiz", auth(http.HandlerFunc(h.getQuiz)))
	mux.Handle("POST /api/resources/{resourceId}/submit-quiz", auth(http.HandlerFunc(h.submitQuiz)))
	mux.Handle("GET /api/resources/{resourceId}/review/{studentId}", auth(http.HandlerFunc(h.getQuizReview)))

	mux.Handle("GET /api/me/profile", auth(http.HandlerFunc(h.getProfile)))
	mux.Handle("GET /api/groups/{id}/students/{studentId}/stats", auth(http.HandlerFunc(h.getStudentStatsForTeacher)))
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &req) {
		return
	}
	g, err := h.svc.CreateGroup(r.Context(), uID, req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (h *Handler) listOwnedGroups(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	gs, err := h.svc.GroupsOwned(r.Context(), uID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, gs)
}

func (h *Handler) listMyGroups(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	gs, err := h.svc.MyGroups(r.Context(), uID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, gs)
}

func (h *Handler) groupDetail(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	g, mems, err := h.svc.GroupDetail(r.Context(), uID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": g.ID, "name": g.Name, "members": mems})
}

func (h *Handler) addMembers(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Emails []string `json:"emails"`
	}
	if !decode(w, r, &req) {
		return
	}
	mems, err := h.svc.AddMembers(r.Context(), uID, r.PathValue("id"), req.Emails)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mems)
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	err := h.svc.RemoveMember(r.Context(), uID, r.PathValue("id"), r.PathValue("memberID"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createSection(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Title    string `json:"title"`
		Position int    `json:"position"`
	}
	if !decode(w, r, &req) {
		return
	}
	sec, err := h.svc.CreateSection(r.Context(), uID, r.PathValue("id"), req.Title, req.Position)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sec)
}

func (h *Handler) updateSection(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Title string `json:"title"`
	}
	if !decode(w, r, &req) {
		return
	}
	err := h.svc.UpdateSection(r.Context(), uID, r.PathValue("sectionId"), req.Title)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) deleteSection(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	err := h.svc.DeleteSection(r.Context(), uID, r.PathValue("sectionId"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createResource(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)
	//nolint:gosec // Limitado de forma segura por MaxBytesReader en la línea superior
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fichero demasiado grande"})
		return
	}

	rType := r.FormValue("type")
	title := r.FormValue("title")
	content := r.FormValue("content")
	dueStr := r.FormValue("due_at")

	var dueAt *time.Time
	if dueStr != "" {
		if t, err := time.Parse(time.RFC3339, dueStr); err == nil {
			dueAt = &t
		}
	}

	filePath := ""
	file, header, err := r.FormFile("file")
	if err == nil {
		defer file.Close()
		path, saveErr := h.svc.SaveUploadedFile(file, header.Filename)
		if saveErr == nil {
			filePath = path
		}
	}

	res, err := h.svc.CreateResource(r.Context(), uID, r.PathValue("sectionId"), rType, title, content, filePath, dueAt)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (h *Handler) createQuiz(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Title     string         `json:"title"`
		Questions []QuizQuestion `json:"questions"`
	}
	if !decode(w, r, &req) {
		return
	}
	res, err := h.svc.CreateQuizWithQuestions(r.Context(), uID, r.PathValue("sectionId"), req.Title, req.Questions)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

// aiGenerateQuiz builds quiz questions with AI from teaching material (a .docx
// file and/or pasted text) plus difficulty, number of questions and focus. It
// does NOT save anything: it returns the generated questions so the teacher can
// review/edit them and then save through the normal create-quiz endpoint.
func (h *Handler) aiGenerateQuiz(w http.ResponseWriter, r *http.Request) {
	if _, ok := authUser(w, r); !ok {
		return
	}

	//nolint:gosec // 10MB cap to bound memory while parsing the upload.
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, errors.New("archivo demasiado grande o formato inválido"))
		return
	}

	content := strings.TrimSpace(r.FormValue("text"))
	if file, fh, err := r.FormFile("file"); err == nil {
		defer file.Close()
		if !strings.EqualFold(filepath.Ext(fh.Filename), ".docx") {
			writeError(w, errors.New("formato no soportado: sube un archivo .docx"))
			return
		}
		extracted, err := notes.ExtractTextFromDocx(file, fh.Size)
		if err != nil {
			writeError(w, fmt.Errorf("no se pudo leer el documento .docx: %w", err))
			return
		}
		if content != "" {
			content += "\n\n"
		}
		content += extracted
	}

	num, _ := strconv.Atoi(r.FormValue("num_questions"))
	questions, err := h.svc.GenerateQuiz(r.Context(), QuizAIParams{
		Content:      content,
		Difficulty:   r.FormValue("difficulty"),
		NumQuestions: num,
		Focus:        r.FormValue("focus"),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"questions": questions})
}

// aiImproveQuiz refines a not-yet-saved set of quiz questions according to a
// free-text instruction (e.g. raise difficulty, level up, rephrase a question).
func (h *Handler) aiImproveQuiz(w http.ResponseWriter, r *http.Request) {
	if _, ok := authUser(w, r); !ok {
		return
	}
	var req struct {
		Questions   []QuizQuestion `json:"questions"`
		Instruction string         `json:"instruction"`
	}
	if !decode(w, r, &req) {
		return
	}
	questions, err := h.svc.ImproveQuiz(r.Context(), req.Questions, req.Instruction)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"questions": questions})
}

func (h *Handler) deleteResource(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	err := h.svc.DeleteResource(r.Context(), uID, r.PathValue("resourceId"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getGroupContent(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	content, err := h.svc.GetGroupContent(r.Context(), uID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, content)
}

func (h *Handler) submitAssignment(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)
	//nolint:gosec // Limitado de forma segura por MaxBytesReader en la línea superior
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fichero excesivo"})
		return
	}

	text := r.FormValue("text_content")
	filePath := ""
	file, header, err := r.FormFile("file")
	if err == nil {
		defer file.Close()
		path, _ := h.svc.SaveUploadedFile(file, header.Filename)
		filePath = path
	}

	err = h.svc.SubmitAssignment(r.Context(), uID, r.PathValue("resourceId"), text, filePath)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) listSubmissions(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	subs, err := h.svc.GetAssignmentSubmissions(r.Context(), uID, r.PathValue("resourceId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

func (h *Handler) gradeSubmission(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Grade    string `json:"grade"`
		Feedback string `json:"feedback"`
	}
	if !decode(w, r, &req) {
		return
	}
	parsedGrade, _ := strconv.ParseFloat(req.Grade, 64)
	err := h.svc.GradeStudentTask(r.Context(), uID, r.PathValue("resourceId"), r.PathValue("studentId"), parsedGrade, req.Feedback)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request) {
	if _, ok := authUser(w, r); !ok {
		return
	}
	safeName := filepath.Clean(filepath.Base(r.PathValue("filename")))
	http.ServeFile(w, r, filepath.Join(".", "uploads", safeName))
}

func (h *Handler) updateResource(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Title, Content string
		DueAt          *time.Time `json:"due_at"`
	}
	if !decode(w, r, &req) {
		return
	}
	err := h.svc.UpdateResource(r.Context(), uID, r.PathValue("resourceId"), req.Title, req.Content, req.DueAt)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) getQuiz(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	quiz, err := h.svc.GetQuiz(r.Context(), uID, r.PathValue("resourceId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, quiz)
}

func (h *Handler) submitQuiz(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Answers map[string]string `json:"answers"`
	}
	if !decode(w, r, &req) {
		return
	}

	grade, err := h.svc.SubmitQuiz(r.Context(), uID, r.PathValue("resourceId"), req.Answers)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"grade": grade})
}

func (h *Handler) getQuizReview(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	review, err := h.svc.GetQuizReview(r.Context(), uID, r.PathValue("resourceId"), r.PathValue("studentId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, review)
}

func authUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := users.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
		return "", false
	}
	return id, true
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request data"})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrGroupNotFound), errors.Is(err, ErrMemberNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	prof, err := h.svc.GetStudentProfile(r.Context(), uID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, prof)
}

func (h *Handler) getStudentStatsForTeacher(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}
	stats, err := h.svc.GetStudentStatsForTeacher(r.Context(), uID, r.PathValue("id"), r.PathValue("studentId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
