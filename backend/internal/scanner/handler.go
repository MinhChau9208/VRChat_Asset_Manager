package scanner

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"vrchat-asset-manager/backend/internal/asset"
)

// Handler serves the scanner endpoints.
type Handler struct {
	svc *Service
	mu  sync.Mutex // one scan at a time
}

// NewHandler creates a scanner handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers scanner routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/scanner/config", h.GetConfig)
	mux.HandleFunc("PUT /api/scanner/config", h.PutConfig)
	mux.HandleFunc("POST /api/scanner/scan", h.Scan)
	mux.HandleFunc("POST /api/scanner/accept", h.Accept)
	mux.HandleFunc("POST /api/scanner/ignore", h.Ignore)
	mux.HandleFunc("GET /api/scanner/ignored", h.ListIgnored)
	mux.HandleFunc("DELETE /api/scanner/ignored", h.Unignore)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// GetConfig handles GET /api/scanner/config
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.svc.LoadConfig(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// PutConfig handles PUT /api/scanner/config
func (h *Handler) PutConfig(w http.ResponseWriter, r *http.Request) {
	var cfg Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json request body: "+err.Error())
		return
	}
	saved, err := h.svc.SaveConfig(r.Context(), cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

// Scan handles POST /api/scanner/scan
func (h *Handler) Scan(w http.ResponseWriter, r *http.Request) {
	if !h.mu.TryLock() {
		writeError(w, http.StatusConflict, "a scan is already running")
		return
	}
	defer h.mu.Unlock()

	result, err := h.svc.Scan(r.Context())
	if errors.Is(err, ErrNoRoots) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type idsRequest struct {
	AssetIDs []int64 `json:"asset_ids"`
}

func (h *Handler) decodeIDs(w http.ResponseWriter, r *http.Request) ([]int64, bool) {
	var req idsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json request body: "+err.Error())
		return nil, false
	}
	if len(req.AssetIDs) == 0 {
		writeError(w, http.StatusBadRequest, "asset_ids is required")
		return nil, false
	}
	return req.AssetIDs, true
}

func writeDraftError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, asset.ErrNotFound):
		writeError(w, http.StatusNotFound, "asset not found")
	case errors.Is(err, ErrNotDraft):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// Accept handles POST /api/scanner/accept
func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	ids, ok := h.decodeIDs(w, r)
	if !ok {
		return
	}
	n, err := h.svc.Accept(r.Context(), ids)
	if err != nil {
		writeDraftError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"accepted": n})
}

// Ignore handles POST /api/scanner/ignore
func (h *Handler) Ignore(w http.ResponseWriter, r *http.Request) {
	ids, ok := h.decodeIDs(w, r)
	if !ok {
		return
	}
	n, err := h.svc.Ignore(r.Context(), ids)
	if err != nil {
		writeDraftError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"ignored": n})
}

// ListIgnored handles GET /api/scanner/ignored
func (h *Handler) ListIgnored(w http.ResponseWriter, r *http.Request) {
	paths, err := h.svc.IgnoredPaths(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, paths)
}

// Unignore handles DELETE /api/scanner/ignored?path=...
func (h *Handler) Unignore(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	if err := h.svc.Unignore(r.Context(), path); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "path will be scanned again"})
}
