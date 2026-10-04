package category

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// Handler serves the category endpoints.
type Handler struct {
	repo *Repository
}

// NewHandler creates a category handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes registers category routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/categories", h.List)
	mux.HandleFunc("POST /api/categories", h.Create)
	mux.HandleFunc("PUT /api/categories/{id}", h.Update)
	mux.HandleFunc("DELETE /api/categories/{id}", h.Delete)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeRepoError maps repository errors to HTTP status codes.
func writeRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrDuplicateName):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrParentNotFound), errors.Is(err, ErrTooDeep), errors.Is(err, ErrHasChildren):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func decodeInput(w http.ResponseWriter, r *http.Request) (Input, bool) {
	var in Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json request body: "+err.Error())
		return in, false
	}
	if strings.TrimSpace(in.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return in, false
	}
	return in, true
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return 0, false
	}
	return id, true
}

// List handles GET /api/categories
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load categories")
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

// Create handles POST /api/categories
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeInput(w, r)
	if !ok {
		return
	}
	created, err := h.repo.Create(r.Context(), in)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// Update handles PUT /api/categories/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	in, ok := decodeInput(w, r)
	if !ok {
		return
	}
	updated, err := h.repo.Update(r.Context(), id, in)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /api/categories/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "category deleted"})
}
