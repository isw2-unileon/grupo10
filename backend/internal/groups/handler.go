package groups

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/isw2-unileon/grupo10/backend/internal/users"
)

// Handler exposes the group endpoints over HTTP. It reuses the users module's
// JWT middleware so every route requires an authenticated user.
type Handler struct {
	svc    *Service
	parser users.TokenParser
}

// NewHandler builds an HTTP handler for the group service.
func NewHandler(svc *Service, parser users.TokenParser) *Handler {
	return &Handler{svc: svc, parser: parser}
}

// RegisterRoutes wires the group endpoints onto the given mux. Every route is
// wrapped in RequireAuth.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	auth := users.RequireAuth(h.parser)
	mux.Handle("POST /api/groups", auth(http.HandlerFunc(h.createGroup)))
	mux.Handle("GET /api/groups", auth(http.HandlerFunc(h.listOwnedGroups)))
	mux.Handle("GET /api/me/groups", auth(http.HandlerFunc(h.listMyGroups)))
	mux.Handle("GET /api/groups/{id}", auth(http.HandlerFunc(h.groupDetail)))
	mux.Handle("POST /api/groups/{id}/members", auth(http.HandlerFunc(h.addMembers)))
	mux.Handle("DELETE /api/groups/{id}/members/{memberID}", auth(http.HandlerFunc(h.removeMember)))
	mux.Handle("POST /api/groups/{id}/tasks", auth(http.HandlerFunc(h.createTask)))
	mux.Handle("GET /api/groups/{id}/tasks", auth(http.HandlerFunc(h.listTasks)))
}

type createGroupRequest struct {
	Name string `json:"name"`
}

type addMembersRequest struct {
	Emails []string `json:"emails"`
}

type createTaskRequest struct {
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	DueAt       *time.Time `json:"due_at"`
}

type groupDetailResponse struct {
	*Group
	Members []Member `json:"members"`
	Tasks   []Task   `json:"tasks"`
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUser(w, r)
	if !ok {
		return
	}
	var req createGroupRequest
	if !decode(w, r, &req) {
		return
	}

	g, err := h.svc.CreateGroup(r.Context(), userID, req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (h *Handler) listOwnedGroups(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUser(w, r)
	if !ok {
		return
	}
	gs, err := h.svc.GroupsOwned(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orEmptyGroups(gs))
}

func (h *Handler) listMyGroups(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUser(w, r)
	if !ok {
		return
	}
	gs, err := h.svc.MyGroups(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}
	// An empty array tells the frontend to show the "waiting for a group" state.
	writeJSON(w, http.StatusOK, orEmptyGroups(gs))
}

func (h *Handler) groupDetail(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUser(w, r)
	if !ok {
		return
	}
	g, members, tasks, err := h.svc.GroupDetail(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, groupDetailResponse{
		Group:   g,
		Members: orEmptyMembers(members),
		Tasks:   orEmptyTasks(tasks),
	})
}

func (h *Handler) addMembers(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUser(w, r)
	if !ok {
		return
	}
	var req addMembersRequest
	if !decode(w, r, &req) {
		return
	}
	members, err := h.svc.AddMembers(r.Context(), userID, r.PathValue("id"), req.Emails)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orEmptyMembers(members))
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUser(w, r)
	if !ok {
		return
	}
	err := h.svc.RemoveMember(r.Context(), userID, r.PathValue("id"), r.PathValue("memberID"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUser(w, r)
	if !ok {
		return
	}
	var req createTaskRequest
	if !decode(w, r, &req) {
		return
	}
	t, err := h.svc.CreateTask(r.Context(), userID, r.PathValue("id"), TaskInput(req))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUser(w, r)
	if !ok {
		return
	}
	tasks, err := h.svc.ListTasks(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orEmptyTasks(tasks))
}

// authUser reads the user ID injected by RequireAuth. It should always be
// present, but guarding keeps the handlers safe if the wiring changes.
func authUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := users.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
		return "", false
	}
	return id, true
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
	case errors.Is(err, ErrValidation):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrGroupNotFound), errors.Is(err, ErrMemberNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}

// JSON marshals a nil slice as null; return [] instead so clients can iterate.
func orEmptyGroups(gs []Group) []Group {
	if gs == nil {
		return []Group{}
	}
	return gs
}

func orEmptyMembers(ms []Member) []Member {
	if ms == nil {
		return []Member{}
	}
	return ms
}

func orEmptyTasks(ts []Task) []Task {
	if ts == nil {
		return []Task{}
	}
	return ts
}
