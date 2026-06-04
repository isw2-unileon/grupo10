package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/isw2-unileon/grupo10/internal/user"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		// En local si no existe el .env queremos que falle, pero en Render
		// las variables se meten desde el panel de control, por lo que no habrá archivo .env.
		// Ponemos un log avisando, pero no bloqueamos el inicio por si acaso.
		log.Println("Aviso: No se pudo cargar el archivo .env, se usarán las variables del sistema")
	}
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

	// 1. Inicializamos el repositorio de Postgres real que acabamos de crear
	userRepo := user.NewPostgresRepository(db)

	// 2. Inicializamos el servicio de autenticación inyectándole ese repositorio
	authService := user.NewAuthService(userRepo)
	userHandler := user.NewHandler(authService) // <-- El nuevo handler que inyecta el servicio

	// 2. Registramos las rutas HTTP
	http.HandleFunc("/health", healthHandler(db))
	http.HandleFunc("/register", userHandler.RegisterHandler) // <-- RUTA REGISTRO
	http.HandleFunc("/login", userHandler.LoginHandler)       // <-- RUTA LOGIN

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

	// MODIFICACIÓN AQUÍ: Ajustamos la ruta para que apunte a donde está de verdad el up.sql
	// Si la carpeta migrations está dentro de server/src/, pon esto:
	migration, err := os.ReadFile("server/migrations/up.sql")
	// NOTA: Si al ejecutarlo te sigue fallando, prueba a cambiarlo por:
	// migration, err := os.ReadFile("server/migrations/up.sql")
	// dependiendo de dónde tengáis guardada la carpeta 'migrations'.
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
