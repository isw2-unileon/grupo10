package calendar

import (
	"testing"
	"time"
)

// ==========================================
// MOCK DEL REPOSITORIO
// ==========================================
// Creamos un repositorio falso que no hace nada en base de datos,
// solo simula que todo va bien para que podamos aislar y testear el Servicio.
type mockRepository struct{}

func (m *mockRepository) CreateEvent(event *Event) error {
	event.ID = "mock-uuid-1234" // Simulamos que Postgres le asignó un ID
	return nil
}

func (m *mockRepository) GetAvailableTutorings() ([]Event, error) {
	return nil, nil // No lo usamos en este test de momento
}

func (m *mockRepository) CreateBooking(booking *Booking) error {
	return nil
}

// ==========================================
// TESTS DEL SERVICIO
// ==========================================

func TestCreateTutoringSlot(t *testing.T) {
	// 1. Preparamos el servicio con nuestro repositorio de mentira
	repo := &mockRepository{}
	svc := NewService(repo)

	// 2. Definimos los casos de prueba (Table-Driven Tests)
	now := time.Now()

	tests := []struct {
		name        string
		startsAt    time.Time
		endsAt      time.Time
		expectedErr error
	}{
		{
			name:        "Éxito: Fechas válidas en el futuro",
			startsAt:    now.Add(24 * time.Hour), // Mañana
			endsAt:      now.Add(25 * time.Hour), // Mañana + 1 hora
			expectedErr: nil,
		},
		{
			name:        "Error: Fecha de fin antes que inicio",
			startsAt:    now.Add(24 * time.Hour),
			endsAt:      now.Add(23 * time.Hour), // Termina una hora antes de empezar
			expectedErr: ErrInvalidDates,
		},
		{
			name:        "Error: Fecha en el pasado",
			startsAt:    now.Add(-24 * time.Hour), // Ayer
			endsAt:      now.Add(-23 * time.Hour),
			expectedErr: ErrPastDate,
		},
	}

	// 3. Ejecutamos los tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateTutoringSlot("profesor-uuid", "Tutoría de Go", tt.startsAt, tt.endsAt)

			// Comprobamos si el error devuelto es el que esperábamos
			if err != tt.expectedErr {
				t.Errorf("Esperaba el error '%v', pero obtuve '%v'", tt.expectedErr, err)
			}
		})
	}
}
