package notes

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"mime/multipart"
	"net/http"
	"strings"

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
	mux.Handle("POST /api/notes/upload", authMiddleware(http.HandlerFunc(h.uploadNote)))

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

// --- NUEVO IMPORTANTE: Función del endpoint para procesar el archivo Word ---
func (h *Handler) uploadNote(w http.ResponseWriter, r *http.Request) {
	authorID, ok := getUserID(w, r)
	if !ok {
		return
	}

	// 1. Limitamos el tamaño del archivo a 10 Megabytes por seguridad
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Archivo demasiado grande o formato inválido", http.StatusBadRequest)
		return
	}

	// 2. Extraemos el título del formulario
	title := r.FormValue("title")
	if title == "" {
		title = "Documento Importado"
	}

	// 3. Obtenemos el archivo enviado desde Vue
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error al leer el archivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 4. Extraemos el texto del .docx
	content, err := extractTextFromDocx(file, fileHeader.Size)
	if err != nil {
		http.Error(w, "No se pudo procesar el Word. Asegúrate de que es un .docx válido: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. Lo guardamos en la base de datos usando tu servicio
	note, err := h.svc.CreateNote(r.Context(), authorID, title, content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 6. Devolvemos el apunte creado al frontend
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(note)
}

// --- FUNCIÓN HELPER: Extrae el texto plano de un archivo ZIP/DOCX ---
func extractTextFromDocx(file multipart.File, size int64) (string, error) {
	// Abrimos el archivo como un ZIP en memoria
	zr, err := zip.NewReader(file, size)
	if err != nil {
		return "", err
	}

	// Buscamos el XML donde Word guarda el texto real
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

	// Parseamos el XML
	decoder := xml.NewDecoder(rc)
	var textBuilder strings.Builder

	// Recorremos las etiquetas XML
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
