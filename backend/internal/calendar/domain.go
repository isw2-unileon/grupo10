package calendar

import (
	"time"
)

// ==========================================
// 1. MODELOS DE DATOS (Nuestras entidades)
// ==========================================

// Event representa un hueco en el calendario (ej. "Tutoría de Programación")
type Event struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`             // Professor ID
	SubjectID   *string   `json:"subject_id,omitempty"` // Puntero porque en SQL permite NULL
	Title       string    `json:"title"`
	Description string    `json:"description"` // ✅ ¡Perfecto!
	Type        string    `json:"type"`        // 'tutoring', 'deadline', 'exam', 'other'
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// Booking representa a un alumno que ha reservado una tutoría
type Booking struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	StudentID string    `json:"student_id"`
	Status    string    `json:"status"` // 'pending', 'confirmed', 'cancelled'
	BookedAt  time.Time `json:"booked_at"`
}

// ==========================================
// 2. PUERTOS (Interfaces de la Arq. Hexagonal)
// ==========================================

// Repository define qué operaciones le vamos a pedir a Postgres.
type Repository interface {
	CreateEvent(event *Event) error
	GetAvailableTutorings() ([]Event, error)
	CreateBooking(booking *Booking) error
}

// Service define los casos de uso (lo que el usuario quiere hacer realmente).
type Service interface {
	// ⚠️ CORREGIDO: Añadido "description string" a la interfaz
	CreateTutoringSlot(ownerID, title, description string, startsAt, endsAt time.Time) (*Event, error)
	ListAvailableTutorings() ([]Event, error)
	BookTutoring(eventID, studentID string) (*Booking, error)
}
