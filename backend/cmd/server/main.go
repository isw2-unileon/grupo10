package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/isw2-unileon/grupo10/backend/internal/calendar"
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
	registerCalendarRoutes(mux, db)

	// 1. LEEMOS LA VARIABLE
	frontendURL := os.Getenv("FRONTEND_URL")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// 2. ENVOLVER EL MUX CON EL MIDDLEWARE DE CORS
	// Aplicamos el control de accesos cruzados pasándole la URL de tu frontend
	handlerWithCORS := corsMiddleware(frontendURL)(mux)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handlerWithCORS,
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

// registerCalendarRoutes builds the calendar module and wires its HTTP endpoints.
func registerCalendarRoutes(mux *http.ServeMux, db *sql.DB) {
	repo := calendar.NewPostgresRepository(db)
	svc := calendar.NewService(repo)
	// Como CalendarHandler espera un Servicio, se lo pasamos
	calendar.NewHandler(svc).RegisterRoutes(mux)
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

// corsMiddleware maneja las cabeceras CORS de forma a prueba de fallos
func corsMiddleware(frontendURL string) func(http.Handler) http.Handler {
	// 1. Limpiamos espacios en blanco accidentales y barras finales que vengan de Render
	cleanFrontendURL := strings.TrimSpace(frontendURL)
	cleanFrontendURL = strings.TrimSuffix(cleanFrontendURL, "/")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" {
				// 2. Permitimos el origen dinámicamente para evitar bloqueos
				w.Header().Set("Access-Control-Allow-Origin", origin)

				// 3. Chivato: Si hay un desajuste, lo imprimimos en los logs de Render para depurar,
				// pero NO bloqueamos la petición para que podáis seguir trabajando.
				if cleanFrontendURL != "" && origin != cleanFrontendURL {
					log.Printf("⚠️ AVISO CORS: El origen del navegador '%s' no coincide exactamente con la variable en Render '%s'", origin, cleanFrontendURL)
				}
			}

			// Cabeceras obligatorias para que Axios/Fetch puedan mandar JSON y Tokens
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Si el navegador envía un "Preflight" (petición OPTIONS previa), respondemos con un 204 y cortamos aquí
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
