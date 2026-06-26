package calendar

import (
	"database/sql"
)

// PostgresRepository implementa la interfaz calendar.Repository
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository crea una nueva instancia del repositorio
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateEvent inserta un nuevo hueco de tutoría o evento en la base de datos
func (r *PostgresRepository) CreateEvent(event *Event) error {
	// ⚠️ AÑADIDO: description en el INSERT y en los VALUES ($4)
	query := `
        INSERT INTO calendar_events (owner_id, subject_id, title, description, type, starts_at, ends_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, created_at`

	// ⚠️ AÑADIDO: event.Description pasado como argumento a QueryRow
	err := r.db.QueryRow(query, event.OwnerID, event.SubjectID, event.Title, event.Description, event.Type, event.StartsAt, event.EndsAt).
		Scan(&event.ID, &event.CreatedAt)

	return err
}

// GetAvailableTutorings busca todas las tutorías futuras que estén disponibles
func (r *PostgresRepository) GetAvailableTutorings() ([]Event, error) {
	// ⚠️ AÑADIDO: description en el SELECT
	query := `
        SELECT id, owner_id, subject_id, title, description, type, starts_at, ends_at, created_at
        FROM calendar_events
        WHERE type = 'tutoring' AND starts_at > NOW()
        ORDER BY starts_at ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event

		// ⚠️ AÑADIDO: &e.Description en el Scan (tiene que estar en el mismo orden que el SELECT)
		err := rows.Scan(&e.ID, &e.OwnerID, &e.SubjectID, &e.Title, &e.Description, &e.Type, &e.StartsAt, &e.EndsAt, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, nil
}

// CreateBooking registra la reserva de un alumno para una tutoría concreta
func (r *PostgresRepository) CreateBooking(booking *Booking) error {
	query := `
        INSERT INTO tutoring_bookings (event_id, student_id, status)
        VALUES ($1, $2, $3)
        RETURNING id, booked_at`

	err := r.db.QueryRow(query, booking.EventID, booking.StudentID, booking.Status).
		Scan(&booking.ID, &booking.BookedAt)

	return err
}
