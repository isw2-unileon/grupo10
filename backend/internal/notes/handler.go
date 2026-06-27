package notes

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/isw2-unileon/grupo10/backend/internal/users"
)

// Handler maneja las peticiones HTTP de los apuntes.
type Handler struct {
	svc *Service
}

// NewHandler crea un nuevo Handler con el servicio de apuntes.
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

// ShareRequest representa la petición para compartir un apunte.
type ShareRequest struct {
	Email   *string `json:"email,omitempty"`
	GroupID *string `json:"group_id,omitempty"`
}

// RegisterRoutes registra todos los endpoints del módulo de apuntes.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /api/notes", authMiddleware(http.HandlerFunc(h.listNotes)))
	mux.Handle("POST /api/notes", authMiddleware(http.HandlerFunc(h.createNote)))
	mux.Handle("POST /api/notes/upload", authMiddleware(http.HandlerFunc(h.uploadNote)))

	mux.Handle("GET /api/notes/shared", authMiddleware(http.HandlerFunc(h.listSharedNotes)))

	mux.Handle("PUT /api/notes/{id}", authMiddleware(http.HandlerFunc(h.updateNote)))
	mux.Handle("DELETE /api/notes/{id}", authMiddleware(http.HandlerFunc(h.deleteNote)))
	mux.Handle("POST /api/notes/{id}/ai-review", authMiddleware(http.HandlerFunc(h.aiReview)))
	mux.Handle("POST /api/notes/{id}/submit", authMiddleware(http.HandlerFunc(h.submitNote)))

	mux.Handle("POST /api/notes/{id}/share", authMiddleware(http.HandlerFunc(h.shareNote)))

	mux.Handle("GET /api/teacher/notes/pending", authMiddleware(http.HandlerFunc(h.listPending)))
	mux.Handle("POST /api/notes/{id}/approve", authMiddleware(http.HandlerFunc(h.approveNote)))
}

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

	if list == nil {
		w.Write([]byte("[]\n"))
		return
	}
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

//nolint:dupl
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

func (h *Handler) uploadNote(w http.ResponseWriter, r *http.Request) {
	authorID, ok := getUserID(w, r)
	if !ok {
		return
	}

	//nolint:gosec // Limitamos a 10MB por seguridad para no agotar la memoria.
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Archivo demasiado grande o formato inválido", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	if title == "" {
		title = "Documento Importado"
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error al leer el archivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !strings.EqualFold(filepath.Ext(fileHeader.Filename), ".docx") {
		http.Error(w, "Formato no soportado. Sube un archivo .docx", http.StatusBadRequest)
		return
	}

	content, err := ExtractTextFromDocx(file, fileHeader.Size)
	if err != nil {
		http.Error(w, "No se pudo procesar el documento. Asegúrate de que es un .docx válido: "+err.Error(), http.StatusInternalServerError)
		return
	}

	note, err := h.svc.CreateNote(r.Context(), authorID, title, content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(note)
}

// ExtractTextFromDocx extrae el texto plano de un archivo ZIP/DOCX. Es público
// para que otros módulos (p. ej. la generación de tests con IA en groups) puedan
// reutilizar el mismo parseo sin duplicarlo.
//
//nolint:gocognit // El parseo de XML requiere un switch anidado complejo.
func ExtractTextFromDocx(file multipart.File, size int64) (string, error) {
	zr, err := zip.NewReader(file, size)
	if err != nil {
		return "", err
	}

	var docFile *zip.File
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			docFile = f
			break
		}
	}

	if docFile == nil {
		return "", errors.New("no se encontró la estructura de un documento Word")
	}

	rc, err := docFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var textBuilder strings.Builder

	for {
		token, err := decoder.Token()
		if err != nil {
			break // Terminamos de leer
		}

		switch element := token.(type) {
		case xml.StartElement:
			if element.Name.Local == "t" { // Etiqueta <w:t> (Texto)
				var text string
				if err := decoder.DecodeElement(&text, &element); err == nil {
					textBuilder.WriteString(text)
				}
			}
		case xml.EndElement:
			if element.Name.Local == "p" { // Etiqueta </w:p> (Fin de párrafo)
				textBuilder.WriteString("\n\n") // Añadimos saltos de línea para que quede bonito
			}
		}
	}

	return strings.TrimSpace(textBuilder.String()), nil
}

// --- NUEVO: Endpoint para compartir un apunte con un email o grupo ---
//
//nolint:dupl
func (h *Handler) shareNote(w http.ResponseWriter, r *http.Request) {
	authorID, ok := getUserID(w, r)
	if !ok {
		return
	}

	noteID := r.PathValue("id")
	var req ShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if err := h.svc.ShareNote(r.Context(), noteID, authorID, req.Email, req.GroupID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// --- NUEVO: Endpoint para listar los apuntes que me han compartido ---
func (h *Handler) listSharedNotes(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	list, err := h.svc.GetSharedNotes(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	if list == nil {
		w.Write([]byte("[]\n"))
		return
	}

	_ = json.NewEncoder(w).Encode(list)
}
