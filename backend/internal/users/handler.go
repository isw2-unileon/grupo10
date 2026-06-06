package users

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Handler exposes the user endpoints over HTTP.
type Handler struct {
	svc    *Service
	parser TokenParser
}

// NewHandler builds an HTTP handler for the user service. The parser is used to
// authenticate requests to protected routes.
func NewHandler(svc *Service, parser TokenParser) *Handler {
	return &Handler{svc: svc, parser: parser}
}

// RegisterRoutes wires the user endpoints onto the given mux. It relies on the
// method-aware routing patterns available since Go 1.22.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/register", h.register)
	mux.HandleFunc("POST /api/login", h.login)
	mux.Handle("GET /api/me", RequireAuth(h.parser)(http.HandlerFunc(h.me)))
}

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

type authResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decode(w, r, &req) {
		return
	}

	u, token, err := h.svc.Register(r.Context(), RegisterInput(req))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: u})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decode(w, r, &req) {
		return
	}

	u, token, err := h.svc.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: u})
}

// me returns the account of the currently authenticated user. RequireAuth has
// already validated the token and put the user ID in the request context.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
		return
	}

	u, err := h.svc.ByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeError maps domain errors to HTTP status codes, hiding internal details.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation), errors.Is(err, ErrInvalidRole):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrEmailTaken):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrInvalidCredentials):
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrUserNotFound):
		// A valid token whose user no longer exists is treated as unauthenticated.
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "user no longer exists"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}
