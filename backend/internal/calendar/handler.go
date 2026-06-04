package calendar

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// CalendarHandler expone los métodos HTTP
type CalendarHandler struct {
	svc Service
}

// NewHandler inyecta el servicio en el handler
func NewHandler(svc Service) *CalendarHandler {
	return &CalendarHandler{svc: svc}
}

// ==========================================
// ESTRUCTURAS DE PETICIÓN (Lo que esperamos recibir en el JSON)
// ==========================================

type CreateEventRequest struct {
	OwnerID  string    `json:"owner_id"` // TODO: En el futuro lo sacaremos del JWT automáticamente
	Title    string    `json:"title"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

type BookTutoringRequest struct {
	EventID   string `json:"event_id"`
	StudentID string `json:"student_id"` // TODO: También lo sacaremos del JWT
}

// ==========================================
// ENDPOINTS
// ==========================================

// CreateTutoringHandler maneja el POST /tutorings
func (h *CalendarHandler) CreateTutoringHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Formato JSON inválido", http.StatusBadRequest) // 400
		return
	}

	// Delegamos toda la lógica compleja a nuestro Servicio
	event, err := h.svc.CreateTutoringSlot(req.OwnerID, req.Title, req.StartsAt, req.EndsAt)
	if err != nil {
		// Comprobamos si es uno de nuestros errores de negocio personalizados
		if errors.Is(err, ErrInvalidDates) || errors.Is(err, ErrPastDate) {
			http.Error(w, err.Error(), http.StatusBadRequest) // 400
			return
		}
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError) // 500
		return
	}

	// Todo ha ido bien, devolvemos un 201 Created y el evento
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(event)
}

// ListTutoringsHandler maneja el GET /tutorings
func (h *CalendarHandler) ListTutoringsHandler(w http.ResponseWriter, r *http.Request) {
	events, err := h.svc.ListAvailableTutorings()
	if err != nil {
		http.Error(w, "Error al obtener las tutorías", http.StatusInternalServerError)
		return
	}

	// Truco pro: Si no hay eventos, Go devuelve `nil`, que en JSON es `null`.
	// Es mejor devolver un array vacío `[]` para que el Frontend no explote.
	if events == nil {
		events = []Event{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200
	_ = json.NewEncoder(w).Encode(events)
}

// BookTutoringHandler maneja el POST /tutorings/book
func (h *CalendarHandler) BookTutoringHandler(w http.ResponseWriter, r *http.Request) {
	var req BookTutoringRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Formato JSON inválido", http.StatusBadRequest)
		return
	}

	booking, err := h.svc.BookTutoring(req.EventID, req.StudentID)
	if err != nil {
		http.Error(w, "Error al procesar la reserva (quizás ya la reservaste)", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201
	_ = json.NewEncoder(w).Encode(booking)
}

// RegisterRoutes engancha los endpoints del calendario en el ServeMux principal
func (h *CalendarHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tutorings", h.ListTutoringsHandler)
	mux.HandleFunc("POST /api/tutorings", h.CreateTutoringHandler)
	mux.HandleFunc("POST /api/tutorings/book", h.BookTutoringHandler)
}
