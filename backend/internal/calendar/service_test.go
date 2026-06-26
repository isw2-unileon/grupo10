package calendar

import (
	"errors" // <-- Añadimos esto para el errorlint
	"testing"
	"time"
)

// ==========================================
// MOCK DEL REPOSITORIO
// ==========================================
type mockRepository struct{}

func (m *mockRepository) CreateEvent(event *Event) error {
	event.ID = "mock-uuid-1234" // Simulate Postgres assigning an ID
	return nil
}

func (m *mockRepository) GetAvailableTutorings() ([]Event, error) {
	return nil, nil // Not used in this test for now
}

func (m *mockRepository) CreateBooking(booking *Booking) error {
	return nil
}

// ==========================================
// TESTS DEL SERVICIO
// ==========================================

func TestCreateTutoringSlot(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)

	now := time.Now()

	tests := []struct {
		name        string
		startsAt    time.Time
		endsAt      time.Time
		expectedErr error
	}{
		{
			name:        "Éxito: Fechas válidas en el futuro",
			startsAt:    now.Add(24 * time.Hour),
			endsAt:      now.Add(25 * time.Hour),
			expectedErr: nil,
		},
		{
			name:        "Error: Fecha de fin antes que inicio",
			startsAt:    now.Add(24 * time.Hour),
			endsAt:      now.Add(23 * time.Hour),
			expectedErr: ErrInvalidDates,
		},
		{
			name:        "Error: Fecha en el pasado",
			startsAt:    now.Add(-24 * time.Hour),
			endsAt:      now.Add(-23 * time.Hour),
			expectedErr: ErrPastDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ⚠️ CORREGIDO: Añadimos la descripción ("Descripción de prueba") como tercer parámetro
			_, err := svc.CreateTutoringSlot("professor-uuid", "Tutoría de Go", "Descripción de prueba", tt.startsAt, tt.endsAt)

			// FIX DEL LINTER: Usar errors.Is en lugar de err != expectedErr
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("Esperaba el error '%v', pero obtuve '%v'", tt.expectedErr, err)
			}
		})
	}
}
