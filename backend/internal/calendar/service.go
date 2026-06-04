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
func (s *calendarService) CreateTutoringSlot(ownerID, title string, startsAt, endsAt time.Time) (*Event, error) {
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
		OwnerID:  ownerID,
		Title:    title,
		Type:     "tutoring", // Forzamos el tipo 'tutoring' por seguridad
		StartsAt: startsAt,
		EndsAt:   endsAt,
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
	// Aquí podríamos añadir lógica de paginación en el futuro
	return s.repo.GetAvailableTutorings()
}

// BookTutoring crea la reserva en estado "pending"
func (s *calendarService) BookTutoring(eventID, studentID string) (*Booking, error) {
	// Montamos la reserva con estado inicial "pending"
	booking := &Booking{
		EventID:   eventID,
		StudentID: studentID,
		Status:    "pending",
	}

	// Al intentar insertar, Postgres comprobará la restricción UNIQUE(event_id, student_id)
	// que añadimos en la base de datos, evitando que el alumno reserve dos veces la misma cita.
	err := s.repo.CreateBooking(booking)
	if err != nil {
		return nil, err
	}

	return booking, nil
}
