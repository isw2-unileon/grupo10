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
	"github.com/isw2-unileon/grupo10/backend/internal/groups"
	"github.com/isw2-unileon/grupo10/backend/internal/notes"
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

	if err := runMigrations(db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(db))

	// The JWT issuer is shared so every module validates tokens the same way.
	issuer := users.NewJWTIssuer(jwtSecret(), tokenTTL)

	// Registramos todos los módulos
	registerUserRoutes(mux, db, issuer)
	registerCalendarRoutes(mux, db)
	registerGroupRoutes(mux, db, issuer)
	registerNotesRoutes(mux, db, issuer) // <-- NUEVO: Registramos las rutas de notas pasándole el parser de JWT

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

func jwtSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Println("WARNING: JWT_SECRET is not set, using an insecure development secret")
		//nolint:gosec
		secret = "dev-insecure-secret"
	}
	return secret
}

func registerUserRoutes(mux *http.ServeMux, db *sql.DB, issuer *users.JWTIssuer) {
	repo := users.NewPostgresRepository(db)
	svc := users.NewService(repo, issuer)
	users.NewHandler(svc, issuer).RegisterRoutes(mux)
}

func registerGroupRoutes(mux *http.ServeMux, db *sql.DB, parser users.TokenParser) {
	repo := groups.NewPostgresRepository(db)
	svc := groups.NewService(repo)
	groups.NewHandler(svc, parser).RegisterRoutes(mux)
}

func registerCalendarRoutes(mux *http.ServeMux, db *sql.DB) {
	repo := calendar.NewPostgresRepository(db)
	svc := calendar.NewService(repo)
	calendar.NewHandler(svc).RegisterRoutes(mux)
}

// NUEVO: Función para ensamblar el módulo de apuntes y protegerlo con JWT
func registerNotesRoutes(mux *http.ServeMux, db *sql.DB, parser users.TokenParser) {
	repo := notes.NewPostgresRepository(db)
	svc := notes.NewService(repo)
	// Pasamos el middleware RequireAuth para que todos los endpoints de notas estén protegidos
	notes.NewHandler(svc).RegisterRoutes(mux, users.RequireAuth(parser))
}

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
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

func corsMiddleware(frontendURL string) func(http.Handler) http.Handler {
	cleanFrontendURL := strings.TrimSpace(frontendURL)
	cleanFrontendURL = strings.TrimSuffix(cleanFrontendURL, "/")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				if cleanFrontendURL != "" && origin != cleanFrontendURL {
					log.Printf("⚠️ AVISO CORS: El origen '%s' no coincide exactamente con '%s'", origin, cleanFrontendURL)
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
