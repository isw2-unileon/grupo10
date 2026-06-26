package calendar

import (
	"errors"
	"time"
)

// Errores de dominio (para que el handler sepa qué HTTP status devolver luego)
var (
	ErrInvalidDates = errors.New("la fecha de fin debe ser posterior a la de inicio")
	ErrPastDate     = errors.New("no puedes crear una tutoría en el pasado")
)

// calendarService es la implementación real de nuestra interfaz Service
type calendarService struct {
	repo Repository
}

// NewService inyecta el repositorio dentro del servicio
func NewService(repo Repository) Service {
	return &calendarService{
		repo: repo,
	}
}

// CreateTutoringSlot valida las fechas y delega la creación al repositorio
// ⚠️ MODIFICADO: Ahora acepta "description string" como tercer parámetro
func (s *calendarService) CreateTutoringSlot(ownerID, title, description string, startsAt, endsAt time.Time) (*Event, error) {
	// Regla de negocio 1: El evento no puede terminar antes de empezar
	if !endsAt.After(startsAt) {
		return nil, ErrInvalidDates
	}

	// Regla de negocio 2: El evento no puede ser en el pasado
	if startsAt.Before(time.Now()) {
		return nil, ErrPastDate
	}

	// Montamos el objeto Event
	event := &Event{
		OwnerID:     ownerID,
		Title:       title,
		Description: description, // ⚠️ AÑADIDO: Mapeamos la descripción al modelo
		Type:        "tutoring",  // Forzamos el tipo 'tutoring' por seguridad
		StartsAt:    startsAt,
		EndsAt:      endsAt,
		// SubjectID se queda a nil por ahora, asumiendo tutorías genéricas
	}

	// Llamamos a la base de datos
	err := s.repo.CreateEvent(event)
	if err != nil {
		return nil, err
	}

	return event, nil
}

// ListAvailableTutorings es un passthrough directo al repositorio
func (s *calendarService) ListAvailableTutorings() ([]Event, error) {
	return s.repo.GetAvailableTutorings()
}

// BookTutoring crea la reserva en estado "pending"
func (s *calendarService) BookTutoring(eventID, studentID string) (*Booking, error) {
	booking := &Booking{
		EventID:   eventID,
		StudentID: studentID,
		Status:    "pending",
	}

	err := s.repo.CreateBooking(booking)
	if err != nil {
		return nil, err
	}

	return booking, nil
}
