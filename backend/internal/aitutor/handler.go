package aitutor

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/isw2-unileon/grupo10/backend/internal/users"
)

// Handler expone las rutas REST del tutor inteligente.
type Handler struct {
	svc    *Service
	parser users.TokenParser
}

// NewHandler crea un nuevo controlador HTTP.
func NewHandler(svc *Service, parser users.TokenParser) *Handler {
	return &Handler{svc: svc, parser: parser}
}

// RegisterRoutes conecta los endpoints en el multiplexor.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	auth := users.RequireAuth(h.parser)
	// Cambiamos el endpoint a un patrón general libre de path values fijos
	mux.Handle("GET /api/ai-quiz", auth(http.HandlerFunc(h.generateQuiz)))
}

func (h *Handler) generateQuiz(w http.ResponseWriter, r *http.Request) {
	uID, ok := authUser(w, r)
	if !ok {
		return
	}

	// Leemos los IDs acumulados en la URL (ej: /api/ai-quiz?sections=id1,id2,id3)
	sectionsParam := r.URL.Query().Get("sections")
	if sectionsParam == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "debes seleccionar al menos un tema para el tutor"})
		return
	}

	sectionIDs := strings.Split(sectionsParam, ",")
	quiz, err := h.svc.GenerateAIQuiz(r.Context(), uID, sectionIDs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "error procesando modelo IA multitemático"})
		return
	}

	writeJSON(w, http.StatusOK, quiz)
}

func authUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := users.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "no autorizado"})
		return "", false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
