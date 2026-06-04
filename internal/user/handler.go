package user

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// Handler contiene el servicio para poder usarlo en los endpoints
type Handler struct {
	service *AuthService
}

// NewHandler es el constructor
func NewHandler(service *AuthService) *Handler {
	return &Handler{service: service}
}

// Estructuras para parsear el JSON de entrada
type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterHandler maneja la petición POST /register
func (h *Handler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	// Llamamos a tu servicio con los datos del JSON
	err := h.service.Register(r.Context(), req.Name, req.Email, req.Password, req.Role)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		// Mapeamos tus errores de negocio a códigos de estado HTTP correctos
		if errors.Is(err, ErrUserAlreadyExists) {
			log.Printf("ERROR CRÍTICO AL REGISTRAR: %v\n", err)
			w.WriteHeader(http.StatusConflict) // 409
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrInvalidRole) {
			w.WriteHeader(http.StatusBadRequest) // 400
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		log.Printf("ERROR CRÍTICO AL REGISTRAR: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}

	// Si todo va bien, devolvemos un 21 Created
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Usuario registrado con éxito"})
}

// LoginHandler maneja la petición POST /login
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	user, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if errors.Is(err, ErrInvalidCredentials) {
			w.WriteHeader(http.StatusUnauthorized) // 401 Unauthorized
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Devolvemos los datos del usuario logueado (Menos la contraseña por seguridad)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.RoleID,
	})
}
