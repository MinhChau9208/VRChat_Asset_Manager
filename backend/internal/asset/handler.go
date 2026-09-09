package asset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// OpenerFunc defines the function signature for opening paths in the OS file explorer.
type OpenerFunc func(path string, isDir bool) error

// Handler handles HTTP requests for asset endpoints.
type Handler struct {
	repo        *Repository
	opener      OpenerFunc
	previewsDir string
}

func defaultPreviewsDir() string {
	if env := os.Getenv("PREVIEWS_DIR"); env != "" {
		return env
	}
	if _, err := os.Stat("../data"); err == nil {
		return filepath.Join("..", "data", "previews")
	}
	return filepath.Join("data", "previews")
}

func defaultOpener(cleanedPath string, isDir bool) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		if isDir {
			cmd = exec.Command("explorer.exe", cleanedPath)
		} else {
			cmd = exec.Command("explorer.exe", "/select,"+cleanedPath)
		}
	case "darwin":
		if isDir {
			cmd = exec.Command("open", cleanedPath)
		} else {
			cmd = exec.Command("open", "-R", cleanedPath)
		}
	default:
		if isDir {
			cmd = exec.Command("xdg-open", cleanedPath)
		} else {
			cmd = exec.Command("xdg-open", filepath.Dir(cleanedPath))
		}
	}
	return cmd.Start()
}

// NewHandler initializes a new asset handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:        repo,
		opener:      defaultOpener,
		previewsDir: defaultPreviewsDir(),
	}
}

// SetOpener overrides the opener function (useful for tests).
func (h *Handler) SetOpener(opener OpenerFunc) {
	h.opener = opener
}

// SetPreviewsDir sets the directory where preview images are stored.
func (h *Handler) SetPreviewsDir(dir string) {
	h.previewsDir = dir
}

// RegisterRoutes registers the asset CRUD and preview routes on the provided HTTP mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/assets", h.List)
	mux.HandleFunc("GET /api/assets/{id}", h.GetByID)
	mux.HandleFunc("GET /api/assets/{id}/status", h.GetStatus)
	mux.HandleFunc("POST /api/assets/{id}/favorite", h.ToggleFavorite)
	mux.HandleFunc("POST /api/assets/batch-status", h.BatchStatus)
	mux.HandleFunc("POST /api/assets/{id}/open-folder", h.OpenFolder)
	mux.HandleFunc("POST /api/assets/{id}/preview", h.UploadPreview)
	mux.HandleFunc("GET /api/assets/{id}/preview", h.GetPreview)
	mux.HandleFunc("DELETE /api/assets/{id}/preview", h.DeletePreview)
	mux.HandleFunc("POST /api/assets", h.Create)
	mux.HandleFunc("PUT /api/assets/{id}", h.Update)
	mux.HandleFunc("DELETE /api/assets/{id}", h.Delete)
	mux.HandleFunc("GET /api/tags", h.GetTags)
	mux.HandleFunc("POST /api/tags", h.CreateTag)
	mux.HandleFunc("POST /api/filesystem/pick-folder", h.PickFolder)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func validateBoothURL(rawURL string) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return nil
	}
	u, err := url.ParseRequestURI(trimmed)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("invalid booth_url: must be a valid http or https url")
	}
	return nil
}

func validateTags(tags []string) error {
	for _, t := range tags {
		if strings.TrimSpace(t) == "" {
			return errors.New("tag name cannot be empty")
		}
	}
	return nil
}

func parseBoolParam(query url.Values, key string) *bool {
	val := strings.TrimSpace(query.Get(key))
	if val == "" {
		return nil
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return nil
	}
	return &b
}

func parseTagsParam(query url.Values) []string {
	var tags []string
	seen := make(map[string]bool)

	addTag := func(t string) {
		t = strings.TrimSpace(t)
		if t != "" && !seen[t] {
			seen[t] = true
			tags = append(tags, t)
		}
	}

	if single := query.Get("tag"); single != "" {
		addTag(single)
	}

	for _, raw := range query["tags"] {
		parts := strings.Split(raw, ",")
		for _, p := range parts {
			addTag(p)
		}
	}

	return tags
}

// List handles GET /api/assets
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	parsedTags := parseTagsParam(query)
	legacyTag := strings.TrimSpace(query.Get("tag"))
	if len(parsedTags) == 0 && legacyTag != "" {
		parsedTags = []string{legacyTag}
	}

	filters := FilterParams{
		Category:    strings.TrimSpace(query.Get("category")),
		Tag:         legacyTag,
		Tags:        parsedTags,
		Search:      strings.TrimSpace(query.Get("search")),
		Favorite:    parseBoolParam(query, "favorite"),
		HasPreview:  parseBoolParam(query, "has_preview"),
		HasBooth:    parseBoolParam(query, "has_booth"),
		LocalStatus: strings.TrimSpace(query.Get("local_status")),
		Sort:        strings.TrimSpace(query.Get("sort")),
	}

	assets, err := h.repo.List(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list assets: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, assets)
}

// ToggleFavorite handles POST /api/assets/{id}/favorite
func (h *Handler) ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	var targetFav *bool
	if r.Body != nil && r.ContentLength > 0 {
		var req ToggleFavoriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.IsFavorite != nil {
			targetFav = req.IsFavorite
		}
	}

	updated, err := h.repo.ToggleFavorite(r.Context(), id, targetFav)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to toggle favorite: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// BatchStatus handles POST /api/assets/batch-status
func (h *Handler) BatchStatus(w http.ResponseWriter, r *http.Request) {
	var req BatchStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json request body: "+err.Error())
		return
	}

	res, err := h.repo.BatchStatus(r.Context(), req.IDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check batch status: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, BatchStatusResponse{Statuses: res})
}

// GetByID handles GET /api/assets/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	asset, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve asset: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, asset)
}

// GetStatus handles GET /api/assets/{id}/status
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	asset, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve asset: "+err.Error())
		return
	}

	trimmedPath := strings.TrimSpace(asset.LocalPath)
	if trimmedPath == "" {
		writeJSON(w, http.StatusOK, map[string]bool{"exists": false})
		return
	}

	_, statErr := os.Stat(trimmedPath)
	exists := statErr == nil

	writeJSON(w, http.StatusOK, map[string]bool{"exists": exists})
}

// OpenFolder handles POST /api/assets/{id}/open-folder
func (h *Handler) OpenFolder(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	asset, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve asset: "+err.Error())
		return
	}

	trimmedPath := strings.TrimSpace(asset.LocalPath)
	if trimmedPath == "" {
		writeError(w, http.StatusBadRequest, "asset has no local path configured")
		return
	}

	cleanedPath := filepath.Clean(trimmedPath)
	fi, statErr := os.Stat(cleanedPath)
	if statErr != nil {
		writeError(w, http.StatusNotFound, "local path does not exist on disk")
		return
	}

	if h.opener == nil {
		h.opener = defaultOpener
	}

	if err := h.opener(cleanedPath, fi.IsDir()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to open folder: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Create handles POST /api/assets
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json request body: "+err.Error())
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := validateBoothURL(req.BoothURL); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validateTags(req.Tags); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := h.repo.Create(r.Context(), req)
	if errors.Is(err, ErrCategoryNotFound) {
		writeError(w, http.StatusBadRequest, "category not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create asset: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

// Update handles PUT /api/assets/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	var req UpdateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json request body: "+err.Error())
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := validateBoothURL(req.BoothURL); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := validateTags(req.Tags); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.repo.Update(r.Context(), id, req)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}
	if errors.Is(err, ErrCategoryNotFound) {
		writeError(w, http.StatusBadRequest, "category not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update asset: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// UploadPreview handles POST /api/assets/{id}/preview
func (h *Handler) UploadPreview(w http.ResponseWriter, r *http.Request) {
	const maxUploadSize = 10 * 1024 * 1024 // 10MB

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	asset, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve asset: "+err.Error())
		return
	}

	// Limit request body to 10MB
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse multipart form or file exceeds 10MB limit: "+err.Error())
		return
	}

	var fileHeader *multipart.FileHeader
	if files := r.MultipartForm.File["file"]; len(files) > 0 {
		fileHeader = files[0]
	} else if files := r.MultipartForm.File["preview"]; len(files) > 0 {
		fileHeader = files[0]
	} else {
		for _, files := range r.MultipartForm.File {
			if len(files) > 0 {
				fileHeader = files[0]
				break
			}
		}
	}

	if fileHeader == nil {
		writeError(w, http.StatusBadRequest, "no image file provided")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read uploaded file: "+err.Error())
		return
	}
	defer file.Close()

	headerBuf := make([]byte, 512)
	n, err := file.Read(headerBuf)
	if err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "failed to inspect file contents")
		return
	}

	detectedType := http.DetectContentType(headerBuf[:n])
	var ext string
	switch detectedType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	default:
		if n >= 12 && string(headerBuf[0:4]) == "RIFF" && string(headerBuf[8:12]) == "WEBP" {
			ext = ".webp"
		} else {
			writeError(w, http.StatusBadRequest, "unsupported file type: only JPEG, PNG, and WebP are allowed")
			return
		}
	}

	if err := os.MkdirAll(h.previewsDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create previews directory: "+err.Error())
		return
	}

	newFilename := fmt.Sprintf("%d%s", id, ext)
	newFilePath := filepath.Join(h.previewsDir, newFilename)

	dst, err := os.Create(newFilePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create preview file: "+err.Error())
		return
	}

	var reader io.Reader
	if seeker, ok := file.(io.Seeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
		reader = file
	} else {
		reader = io.MultiReader(bytes.NewReader(headerBuf[:n]), file)
	}

	if _, err := io.Copy(dst, reader); err != nil {
		_ = dst.Close()
		_ = os.Remove(newFilePath)
		writeError(w, http.StatusInternalServerError, "failed to save preview file: "+err.Error())
		return
	}
	_ = dst.Close()

	dbPreviewPath := fmt.Sprintf("data/previews/%s", newFilename)
	if err := h.repo.UpdatePreviewPath(r.Context(), id, dbPreviewPath); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update preview path in database: "+err.Error())
		return
	}

	// If old preview had a different filename/extension, clean it up
	if asset.PreviewPath != "" && asset.PreviewPath != dbPreviewPath {
		oldBase := filepath.Base(asset.PreviewPath)
		if oldBase != newFilename && strings.HasPrefix(oldBase, fmt.Sprintf("%d.", id)) {
			_ = os.Remove(filepath.Join(h.previewsDir, oldBase))
		}
	}

	updatedAsset, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reload asset: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updatedAsset)
}

// GetPreview handles GET /api/assets/{id}/preview
func (h *Handler) GetPreview(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	asset, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve asset: "+err.Error())
		return
	}

	trimmedPath := strings.TrimSpace(asset.PreviewPath)
	if trimmedPath == "" {
		writeError(w, http.StatusNotFound, "preview not found for this asset")
		return
	}

	previewFilename := filepath.Base(trimmedPath)
	targetPath := filepath.Join(h.previewsDir, previewFilename)
	cleanTarget := filepath.Clean(targetPath)
	cleanDir := filepath.Clean(h.previewsDir)

	if !strings.HasPrefix(cleanTarget, cleanDir) {
		writeError(w, http.StatusBadRequest, "invalid preview path")
		return
	}

	fi, err := os.Stat(cleanTarget)
	if err != nil || fi.IsDir() {
		// Fallback check if trimmedPath exists directly
		if fi2, err2 := os.Stat(trimmedPath); err2 == nil && !fi2.IsDir() {
			cleanTarget = filepath.Clean(trimmedPath)
		} else {
			writeError(w, http.StatusNotFound, "preview file does not exist on disk")
			return
		}
	}

	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, cleanTarget)
}

// DeletePreview handles DELETE /api/assets/{id}/preview
func (h *Handler) DeletePreview(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	asset, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve asset: "+err.Error())
		return
	}

	if asset.PreviewPath != "" {
		previewFilename := filepath.Base(asset.PreviewPath)
		targetPath := filepath.Join(h.previewsDir, previewFilename)
		// Safety check: ensure targetPath is strictly inside previewsDir
		if rel, err := filepath.Rel(h.previewsDir, targetPath); err == nil && !strings.HasPrefix(rel, "..") {
			_ = os.Remove(targetPath)
		}
		_ = h.repo.UpdatePreviewPath(r.Context(), id, "")
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "preview deleted"})
}

// Delete handles DELETE /api/assets/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	// Clean up application-owned preview image file associated with this asset.
	// CRITICAL SAFETY REQUIREMENT:
	// Only files inside h.previewsDir may ever be deleted.
	// Never delete, modify, rename, or touch local_path or any files outside h.previewsDir.
	if existing, err := h.repo.GetByID(r.Context(), id); err == nil && existing != nil && existing.PreviewPath != "" {
		filename := filepath.Base(existing.PreviewPath)
		targetPath := filepath.Join(h.previewsDir, filename)
		if rel, err := filepath.Rel(h.previewsDir, targetPath); err == nil && !strings.HasPrefix(rel, "..") {
			_ = os.Remove(targetPath)
		}
	}

	err = h.repo.Delete(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete asset: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "asset deleted"})
}

// GetTags handles GET /api/tags
func (h *Handler) GetTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.repo.GetTags(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load tags: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

// CreateTag handles POST /api/tags
func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json request body: "+err.Error())
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "tag name is required")
		return
	}

	tag, err := h.repo.CreateTag(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create tag: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, tag)
}

// PickFolder handles POST /api/filesystem/pick-folder
func (h *Handler) PickFolder(w http.ResponseWriter, r *http.Request) {
	if runtime.GOOS != "windows" {
		writeJSON(w, http.StatusOK, map[string]string{"path": ""})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	psScript := `Add-Type -AssemblyName System.Windows.Forms; $f = New-Object System.Windows.Forms.FolderBrowserDialog; $f.ShowNewFolderButton = $false; $f.Description = 'Select VRChat Asset Folder'; if ($f.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { [Console]::Out.Write($f.SelectedPath) }`

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"path": ""})
		return
	}

	selectedPath := strings.TrimSpace(out.String())
	writeJSON(w, http.StatusOK, map[string]string{"path": selectedPath})
}
