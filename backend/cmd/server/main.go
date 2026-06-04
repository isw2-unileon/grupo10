package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/isw2-unileon/grupo10/backend/internal/users"
	_ "github.com/lib/pq"
)

const (
	tokenTTL     = 24 * time.Hour
	readTimeout  = 15 * time.Second
	writeTimeout = 15 * time.Second
	idleTimeout  = 60 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run wires up the server and blocks until it stops. Returning an error instead
// of calling log.Fatal directly lets deferred cleanup (db.Close) run on exit.
func run() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("missing DATABASE_URL environment variable")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("open database connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	log.Println("Successfully connected to the database")

	// Run migrations automatically on startup.
	if err := runMigrations(db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(db))
	registerUserRoutes(mux, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
	log.Printf("Server listening on port %s", port)
	return srv.ListenAndServe()
}

// registerUserRoutes builds the users module and wires its HTTP endpoints.
func registerUserRoutes(mux *http.ServeMux, db *sql.DB) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Println("WARNING: JWT_SECRET is not set, using an insecure development secret")
		//nolint:gosec // G101: not a real credential, just a dev fallback; production reads JWT_SECRET from the env
		secret = "dev-insecure-secret"
	}

	repo := users.NewPostgresRepository(db)
	issuer := users.NewJWTIssuer(secret, tokenTTL)
	svc := users.NewService(repo, issuer)
	users.NewHandler(svc, issuer).RegisterRoutes(mux)
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

// healthHandler returns 200 if the server and DB are up.
func healthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}
}
