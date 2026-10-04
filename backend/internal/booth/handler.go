package booth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"vrchat-asset-manager/backend/internal/asset"
)

// Handler serves the BOOTH import endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a BOOTH handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers BOOTH routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/booth/lookup", h.Lookup)
	mux.HandleFunc("POST /api/booth/preview", h.Preview)
	mux.HandleFunc("POST /api/booth/apply", h.Apply)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway // BOOTH unreachable or unexpected
	switch {
	case errors.Is(err, ErrInvalidURL), errors.Is(err, ErrImageHost), errors.Is(err, asset.ErrUnsupportedImage):
		status = http.StatusBadRequest
	case errors.Is(err, ErrItemNotFound), errors.Is(err, asset.ErrNotFound):
		status = http.StatusNotFound
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// Lookup handles GET /api/booth/lookup?url=...&refresh=1
func (h *Handler) Lookup(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	input := strings.TrimSpace(q.Get("url"))
	if input == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url is required"})
		return
	}
	suggestion, err := h.svc.Lookup(r.Context(), input, q.Get("refresh") == "1")
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, suggestion)
}

// Preview handles POST /api/booth/preview {"asset_id": 1, "url": "https://booth.pximg.net/..."}
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AssetID int64  `json:"asset_id"`
		URL     string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AssetID == 0 || req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "asset_id and url are required"})
		return
	}
	updated, err := h.svc.SetPreviewFromURL(r.Context(), req.AssetID, req.URL)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// Apply handles POST /api/booth/apply {"asset_ids": [...], "include_tags": false}
func (h *Handler) Apply(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AssetIDs    []int64 `json:"asset_ids"`
		IncludeTags bool    `json:"include_tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.AssetIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "asset_ids is required"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": h.svc.Apply(r.Context(), req.AssetIDs, req.IncludeTags)})
}
