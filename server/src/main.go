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
	try{
		db, err := sql.Open("postgres", dbURL)
		if err != nil {
			log.Fatalf("Failed to open database connection: %v", err)
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			log.Fatalf("Failed to ping database: %v", err)
		}
		log.Println("Successfully connected to the database")

		// Run migrations automatically on startup
		if err := runMigrations(db); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}

		http.HandleFunc("/health", healthHandler(db))

		port := ":8080"
		log.Printf("Server listening on %s", port)
		if err := http.ListenAndServe(port, nil); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}catch{
		log.Fatalf("An unexpected error occurred: %v", err)
	}
	
}

// runMigrations reads up.sql from disk and executes it against the database.
// Using IF NOT EXISTS makes it safe to run multiple times without errors.
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
			json.NewEncoder(w).Encode(map[string]string{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}
}
