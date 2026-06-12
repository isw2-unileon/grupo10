package notes

import (
	"encoding/json"
	"net/http"

	"github.com/isw2-unileon/grupo10/backend/internal/users"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type noteRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type feedbackRequest struct {
	Feedback string `json:"feedback"`
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /api/notes", authMiddleware(http.HandlerFunc(h.listNotes)))
	mux.Handle("POST /api/notes", authMiddleware(http.HandlerFunc(h.createNote)))
	mux.Handle("PUT /api/notes/{id}", authMiddleware(http.HandlerFunc(h.updateNote)))
	mux.Handle("DELETE /api/notes/{id}", authMiddleware(http.HandlerFunc(h.deleteNote)))

	mux.Handle("POST /api/notes/{id}/ai-review", authMiddleware(http.HandlerFunc(h.aiReview)))
	mux.Handle("POST /api/notes/{id}/submit", authMiddleware(http.HandlerFunc(h.submitNote)))

	mux.Handle("GET /api/teacher/notes/pending", authMiddleware(http.HandlerFunc(h.listPending)))
	mux.Handle("POST /api/notes/{id}/approve", authMiddleware(http.HandlerFunc(h.approveNote)))
}

// Función auxiliar para sacar el ID real del usuario desde el JWT Token
func getUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	authorID, ok := users.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "No autenticado", http.StatusUnauthorized)
	}
	return authorID, ok
}

func (h *Handler) listNotes(w http.ResponseWriter, r *http.Request) {
	authorID, ok := getUserID(w, r)
	if !ok {
		return
	}

	list, err := h.svc.GetNotesByAuthor(r.Context(), authorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (h *Handler) createNote(w http.ResponseWriter, r *http.Request) {
	authorID, ok := getUserID(w, r)
	if !ok {
		return
	}

	var req noteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	note, err := h.svc.CreateNote(r.Context(), authorID, req.Title, req.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(note)
}

func (h *Handler) updateNote(w http.ResponseWriter, r *http.Request) {
	authorID, ok := getUserID(w, r)
	if !ok {
		return
	}

	noteID := r.PathValue("id")
	var req noteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if err := h.svc.UpdateNote(r.Context(), noteID, authorID, req.Title, req.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) deleteNote(w http.ResponseWriter, r *http.Request) {
	authorID, ok := getUserID(w, r)
	if !ok {
		return
	}

	noteID := r.PathValue("id")
	if err := h.svc.DeleteNote(r.Context(), noteID, authorID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) aiReview(w http.ResponseWriter, r *http.Request) {
	noteID := r.PathValue("id")
	note, err := h.svc.repo.GetByID(r.Context(), noteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := h.svc.RequestAIReview(r.Context(), note.ID, note.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) submitNote(w http.ResponseWriter, r *http.Request) {
	noteID := r.PathValue("id")
	if err := h.svc.SubmitForApproval(r.Context(), noteID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) listPending(w http.ResponseWriter, r *http.Request) {
	notes, err := h.svc.GetPendingForTeacher(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(notes)
}

func (h *Handler) approveNote(w http.ResponseWriter, r *http.Request) {
	noteID := r.PathValue("id")
	var req feedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if err := h.svc.ApproveNoteWithFeedback(r.Context(), noteID, req.Feedback); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
