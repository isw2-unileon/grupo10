package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}

	// SOLUCIÓN: El defer asegura que la base de datos se cierre SOLO cuando
	// la función main termine por completo (al apagar el servidor).
	defer db.Close()

	// Ahora el Ping funcionará perfectamente porque la conexión sigue abierta.
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to the database")

	// Run migrations automatically on startup
	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	http.HandleFunc("/health", healthHandler(db))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Puerto por defecto para local
	}
	log.Printf("Server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// runMigrations reads up.sql from disk and executes it against the database.
func runMigrations(db *sql.DB) error {
	log.Println("Running migrations...")

	migration, err := os.ReadFile("migrations/up.sql")
	if err != nil {
		return err
	}

	if _, err := db.Exec(string(migration)); err != nil {
		return err
	}

	log.Println("Migrations applied successfully")
	return nil
}

// healthHandler returns 200 if the server and DB are up
func healthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			// json.NewEncoder se usa directamente ya que no devuelve error al inicializarse
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}

		// Estado OK corregido sin el condicional que rompía el flujo
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}
}
