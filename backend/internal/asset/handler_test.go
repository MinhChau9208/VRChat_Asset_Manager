package asset_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"vrchat-asset-manager/backend/internal/asset"
	"vrchat-asset-manager/backend/internal/database"
	"vrchat-asset-manager/backend/migrations"
)

func setupTestServer(t *testing.T) (*http.ServeMux, func()) {
	mux, _, cleanup := setupTestServerWithConfig(t, nil)
	return mux, cleanup
}

func setupTestServerWithOpener(t *testing.T, opener asset.OpenerFunc) (*http.ServeMux, func()) {
	mux, _, cleanup := setupTestServerWithConfig(t, opener)
	return mux, cleanup
}

func setupTestServerWithConfig(t *testing.T, opener asset.OpenerFunc) (*http.ServeMux, string, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "vram-asset-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.Connect(dbPath)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		t.Fatalf("failed to connect test db: %v", err)
	}

	if err := db.Migrate(migrations.FS); err != nil {
		_ = db.Close()
		_ = os.RemoveAll(tempDir)
		t.Fatalf("failed to run migrations: %v", err)
	}

	previewsDir := filepath.Join(tempDir, "previews")
	_ = os.MkdirAll(previewsDir, 0755)

	repo := asset.NewRepository(db.DB)
	handler := asset.NewHandler(repo)
	handler.SetPreviewsDir(previewsDir)
	if opener != nil {
		handler.SetOpener(opener)
	}

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	cleanup := func() {
		_ = db.Close()
		_ = os.RemoveAll(tempDir)
	}

	return mux, previewsDir, cleanup
}

func doMultipartUpload(mux *http.ServeMux, target, fieldName, filename string, content []byte) *httptest.ResponseRecorder {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		panic(err)
	}
	_, _ = part.Write(content)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func doRequest(mux *http.ServeMux, method, target string, body any) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, target, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// 1. Create Asset
func TestCreateAsset(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	catID := int64(1) // Avatar
	payload := asset.CreateAssetRequest{
		Name:        "Manuka Avatar",
		CategoryID:  &catID,
		Author:      "Jingo Channel",
		BoothURL:    "https://booth.pm/en/items/4394473",
		LocalPath:   "D:/VRChat/Avatars/Manuka",
		Description: "Base model for Manuka",
		Tags:        []string{"avatar", "female"},
	}

	w := doRequest(mux, http.MethodPost, "/api/assets", payload)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var created asset.Asset
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if created.ID == 0 {
		t.Error("Expected valid ID, got 0")
	}
	if created.Name != payload.Name {
		t.Errorf("Expected name %q, got %q", payload.Name, created.Name)
	}
	if len(created.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(created.Tags))
	}
	if created.Category == nil || created.Category.Name != "Avatar" {
		t.Errorf("Expected category 'Avatar', got %v", created.Category)
	}
}

// 2. Get Asset by ID
func TestGetAssetByID(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	catID := int64(2) // Hair
	payload := asset.CreateAssetRequest{
		Name:       "Twin Tail Hair",
		CategoryID: &catID,
		Tags:       []string{"hair", "cute"},
	}

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", payload)
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	wGet := doRequest(mux, http.MethodGet, "/api/assets/"+strconvFormat(created.ID), nil)
	if wGet.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", wGet.Code)
	}

	var fetched asset.Asset
	_ = json.NewDecoder(wGet.Body).Decode(&fetched)
	if fetched.ID != created.ID || fetched.Name != "Twin Tail Hair" {
		t.Errorf("Mismatch fetched asset: %+v", fetched)
	}
}

// 3. List Assets (with search and filter)
func TestListAssets(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	catHair := int64(2)
	catClothes := int64(3)

	_ = doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:       "Gothic Dress",
		CategoryID: &catClothes,
		Tags:       []string{"gothic", "dress"},
	})
	_ = doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:       "Gothic Hairpin",
		CategoryID: &catHair,
		Tags:       []string{"gothic", "accessory"},
	})
	_ = doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:       "Casual Hoodie",
		CategoryID: &catClothes,
		Tags:       []string{"casual"},
	})

	// List all
	wAll := doRequest(mux, http.MethodGet, "/api/assets", nil)
	var all []asset.Asset
	_ = json.NewDecoder(wAll.Body).Decode(&all)
	if len(all) != 3 {
		t.Fatalf("Expected 3 assets, got %d", len(all))
	}

	// Filter by search
	wSearch := doRequest(mux, http.MethodGet, "/api/assets?search=Gothic", nil)
	var searchResult []asset.Asset
	_ = json.NewDecoder(wSearch.Body).Decode(&searchResult)
	if len(searchResult) != 2 {
		t.Fatalf("Expected 2 assets matching search 'Gothic', got %d", len(searchResult))
	}

	// Filter by category name
	wCat := doRequest(mux, http.MethodGet, "/api/assets?category=Clothes", nil)
	var catResult []asset.Asset
	_ = json.NewDecoder(wCat.Body).Decode(&catResult)
	if len(catResult) != 2 {
		t.Fatalf("Expected 2 assets matching category 'Clothes', got %d", len(catResult))
	}

	// Filter by tag
	wTag := doRequest(mux, http.MethodGet, "/api/assets?tag=dress", nil)
	var tagResult []asset.Asset
	_ = json.NewDecoder(wTag.Body).Decode(&tagResult)
	if len(tagResult) != 1 {
		t.Fatalf("Expected 1 asset matching tag 'dress', got %d", len(tagResult))
	}
}

// 4. Update Asset
func TestUpdateAsset(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	catID := int64(1)
	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:       "Initial Name",
		CategoryID: &catID,
		Author:     "Old Author",
		Tags:       []string{"old_tag"},
	})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	updatePayload := asset.UpdateAssetRequest{
		Name:        "Updated Name",
		CategoryID:  &catID,
		Author:      "New Author",
		BoothURL:    "https://booth.pm/en/items/9999",
		Description: "Updated description",
		Tags:        []string{"new_tag", "updated_tag"},
	}

	wUpdate := doRequest(mux, http.MethodPut, "/api/assets/"+strconvFormat(created.ID), updatePayload)
	if wUpdate.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on update, got %d: %s", wUpdate.Code, wUpdate.Body.String())
	}

	var updated asset.Asset
	_ = json.NewDecoder(wUpdate.Body).Decode(&updated)
	if updated.Name != "Updated Name" || updated.Author != "New Author" {
		t.Errorf("Asset fields were not updated properly: %+v", updated)
	}
	if len(updated.Tags) != 2 || updated.Tags[0] != "new_tag" {
		t.Errorf("Asset tags were not updated properly: %+v", updated.Tags)
	}
}

// 5. Delete Asset
func TestDeleteAsset(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name: "Asset To Delete",
		Tags: []string{"tag1", "tag2"},
	})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	wDelete := doRequest(mux, http.MethodDelete, "/api/assets/"+strconvFormat(created.ID), nil)
	if wDelete.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on delete, got %d", wDelete.Code)
	}

	// Verify it no longer exists
	wGet := doRequest(mux, http.MethodGet, "/api/assets/"+strconvFormat(created.ID), nil)
	if wGet.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found after delete, got %d", wGet.Code)
	}
}

// 6. 404 for Missing Asset
func TestGetNonExistentAsset(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	w := doRequest(mux, http.MethodGet, "/api/assets/99999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found, got %d", w.Code)
	}

	var errResp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&errResp)
	if errResp["error"] != "asset not found" {
		t.Errorf("Expected 'asset not found' error, got %q", errResp["error"])
	}
}

// 7. Invalid Category
func TestInvalidCategory(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	invalidCatID := int64(99999)
	payload := asset.CreateAssetRequest{
		Name:       "Asset With Invalid Category",
		CategoryID: &invalidCatID,
	}

	w := doRequest(mux, http.MethodPost, "/api/assets", payload)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for invalid category, got %d", w.Code)
	}

	var errResp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&errResp)
	if errResp["error"] != "category not found" {
		t.Errorf("Expected 'category not found' error, got %q", errResp["error"])
	}
}

// 8. Tag Assignment
func TestTagAssignment(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	payload := asset.CreateAssetRequest{
		Name: "Tagged Asset",
		Tags: []string{"physbone", "quest", "pc"},
	}

	w := doRequest(mux, http.MethodPost, "/api/assets", payload)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d", w.Code)
	}

	var created asset.Asset
	_ = json.NewDecoder(w.Body).Decode(&created)

	if len(created.Tags) != 3 {
		t.Fatalf("Expected 3 tags, got %d", len(created.Tags))
	}
}

// 9. Tag Update / Removal
func TestTagUpdateRemoval(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name: "Asset With Tags",
		Tags: []string{"keep", "remove1", "remove2"},
	})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	// Update with only "keep" and "added"
	wUpdate := doRequest(mux, http.MethodPut, "/api/assets/"+strconvFormat(created.ID), asset.UpdateAssetRequest{
		Name: "Asset With Tags",
		Tags: []string{"keep", "added"},
	})
	var updated asset.Asset
	_ = json.NewDecoder(wUpdate.Body).Decode(&updated)

	if len(updated.Tags) != 2 {
		t.Fatalf("Expected 2 tags after update, got %d: %v", len(updated.Tags), updated.Tags)
	}
	expectedMap := map[string]bool{"keep": true, "added": true}
	for _, tag := range updated.Tags {
		if !expectedMap[tag] {
			t.Errorf("Unexpected tag %q present after update", tag)
		}
	}
}

// 10. Duplicate Tag Prevention
func TestDuplicateTagPrevention(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	payload := asset.CreateAssetRequest{
		Name: "Asset With Duplicate Tags",
		Tags: []string{"Anime", "anime", " ANIME ", "gothic", "Gothic"},
	}

	w := doRequest(mux, http.MethodPost, "/api/assets", payload)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d", w.Code)
	}

	var created asset.Asset
	_ = json.NewDecoder(w.Body).Decode(&created)

	if len(created.Tags) != 2 {
		t.Fatalf("Expected exactly 2 deduplicated tags, got %d: %v", len(created.Tags), created.Tags)
	}
}

// 11. Get Asset Status - Exists
func TestGetAssetStatus_Exists(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "vram-status-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:      "Asset with existing path",
		LocalPath: tempDir,
	})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	w := doRequest(mux, http.MethodGet, "/api/assets/"+strconvFormat(created.ID)+"/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var statusResp map[string]bool
	_ = json.NewDecoder(w.Body).Decode(&statusResp)
	if !statusResp["exists"] {
		t.Errorf("Expected exists: true, got false")
	}
}

// 12. Get Asset Status - Not Exists
func TestGetAssetStatus_NotExists(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	missingPath := filepath.Join(os.TempDir(), "vram-nonexistent-path-123456")

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:      "Asset with missing path",
		LocalPath: missingPath,
	})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	w := doRequest(mux, http.MethodGet, "/api/assets/"+strconvFormat(created.ID)+"/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var statusResp map[string]bool
	_ = json.NewDecoder(w.Body).Decode(&statusResp)
	if statusResp["exists"] {
		t.Errorf("Expected exists: false, got true")
	}
}

// 13. Get Asset Status - Empty Path
func TestGetAssetStatus_EmptyPath(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:      "Asset with empty path",
		LocalPath: "",
	})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	w := doRequest(mux, http.MethodGet, "/api/assets/"+strconvFormat(created.ID)+"/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var statusResp map[string]bool
	_ = json.NewDecoder(w.Body).Decode(&statusResp)
	if statusResp["exists"] {
		t.Errorf("Expected exists: false, got true")
	}
}

// 14. Get Asset Status - 404 for Missing Asset
func TestGetAssetStatus_NotFound(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	w := doRequest(mux, http.MethodGet, "/api/assets/99999/status", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found, got %d", w.Code)
	}
}

// 15. Open Folder - Success
func TestOpenFolder_Success(t *testing.T) {
	var openedPath string
	var openedIsDir bool
	mockOpener := func(path string, isDir bool) error {
		openedPath = path
		openedIsDir = isDir
		return nil
	}

	mux, cleanup := setupTestServerWithOpener(t, mockOpener)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "vram-open-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:      "Asset to open",
		LocalPath: tempDir,
	})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	w := doRequest(mux, http.MethodPost, "/api/assets/"+strconvFormat(created.ID)+"/open-folder", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %q", resp["status"])
	}

	if openedPath != filepath.Clean(tempDir) {
		t.Errorf("Expected opened path %q, got %q", filepath.Clean(tempDir), openedPath)
	}
	if !openedIsDir {
		t.Errorf("Expected openedIsDir true, got false")
	}
}

// 16. Open Folder - Non-Existent Path
func TestOpenFolder_NonExistentPath(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	missingPath := filepath.Join(os.TempDir(), "vram-open-missing-12345")
	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:      "Asset with missing dir",
		LocalPath: missingPath,
	})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	w := doRequest(mux, http.MethodPost, "/api/assets/"+strconvFormat(created.ID)+"/open-folder", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found, got %d: %s", w.Code, w.Body.String())
	}
}

// 17. Open Folder - Empty Path
func TestOpenFolder_EmptyPath(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:      "Asset without path",
		LocalPath: "",
	})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	w := doRequest(mux, http.MethodPost, "/api/assets/"+strconvFormat(created.ID)+"/open-folder", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request, got %d: %s", w.Code, w.Body.String())
	}
}

// 18. Open Folder - 404 for Missing Asset
func TestOpenFolder_NotFound(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	w := doRequest(mux, http.MethodPost, "/api/assets/99999/open-folder", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found, got %d", w.Code)
	}
}

// 19. Upload Preview - Valid JPEG
func TestUploadPreview_JPEG(t *testing.T) {
	mux, previewsDir, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Asset for JPEG"})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	jpegHeader := []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00\xFF\xDB\x00C\x00sample jpeg data")
	wUpload := doMultipartUpload(mux, "/api/assets/"+strconvFormat(created.ID)+"/preview", "file", "avatar.jpg", jpegHeader)
	if wUpload.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", wUpload.Code, wUpload.Body.String())
	}

	var updated asset.Asset
	_ = json.NewDecoder(wUpload.Body).Decode(&updated)
	if !strings.HasSuffix(updated.PreviewPath, ".jpg") {
		t.Errorf("Expected preview path ending in .jpg, got %q", updated.PreviewPath)
	}

	// Verify file exists on disk
	expectedFile := filepath.Join(previewsDir, strconvFormat(created.ID)+".jpg")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("Expected file %s to exist on disk: %v", expectedFile, err)
	}
}

// 20. Upload Preview - Valid PNG
func TestUploadPreview_PNG(t *testing.T) {
	mux, previewsDir, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Asset for PNG"})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	pngHeader := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82")
	wUpload := doMultipartUpload(mux, "/api/assets/"+strconvFormat(created.ID)+"/preview", "preview", "icon.png", pngHeader)
	if wUpload.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", wUpload.Code, wUpload.Body.String())
	}

	expectedFile := filepath.Join(previewsDir, strconvFormat(created.ID)+".png")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("Expected file %s to exist on disk: %v", expectedFile, err)
	}
}

// 21. Upload Preview - Valid WebP
func TestUploadPreview_WebP(t *testing.T) {
	mux, previewsDir, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Asset for WebP"})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	webpHeader := []byte("RIFF\x1a\x00\x00\x00WEBPVP8 \x0e\x00\x00\x000\x01\x00\x9d\x01*\x01\x00\x01\x00\x00")
	wUpload := doMultipartUpload(mux, "/api/assets/"+strconvFormat(created.ID)+"/preview", "file", "model.webp", webpHeader)
	if wUpload.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", wUpload.Code, wUpload.Body.String())
	}

	expectedFile := filepath.Join(previewsDir, strconvFormat(created.ID)+".webp")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("Expected file %s to exist on disk: %v", expectedFile, err)
	}
}

// 22. Upload Preview - Reject Unsupported File Type
func TestUploadPreview_UnsupportedType(t *testing.T) {
	mux, _, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Asset for Text"})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	textContent := []byte("Hello, this is not an image file at all!")
	wUpload := doMultipartUpload(mux, "/api/assets/"+strconvFormat(created.ID)+"/preview", "file", "text.txt", textContent)
	if wUpload.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request, got %d: %s", wUpload.Code, wUpload.Body.String())
	}
}

// 23. Upload Preview - Reject Oversized File (>10MB)
func TestUploadPreview_Oversized(t *testing.T) {
	mux, _, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Asset for Huge File"})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	oversizedContent := make([]byte, 11*1024*1024)
	copy(oversizedContent, []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00\xFF\xDB\x00C\x00"))

	wUpload := doMultipartUpload(mux, "/api/assets/"+strconvFormat(created.ID)+"/preview", "file", "huge.jpg", oversizedContent)
	if wUpload.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for oversized file, got %d: %s", wUpload.Code, wUpload.Body.String())
	}
}

// 24. Upload Preview - 404 for Missing Asset
func TestUploadPreview_NotFound(t *testing.T) {
	mux, _, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	jpegHeader := []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00\xFF\xDB\x00C\x00test")
	wUpload := doMultipartUpload(mux, "/api/assets/99999/preview", "file", "avatar.jpg", jpegHeader)
	if wUpload.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found, got %d", wUpload.Code)
	}
}

// 25. Retrieve Existing Preview
func TestGetPreview_Success(t *testing.T) {
	mux, _, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Asset for Retrieval"})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	pngHeader := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82")
	_ = doMultipartUpload(mux, "/api/assets/"+strconvFormat(created.ID)+"/preview", "file", "photo.png", pngHeader)

	wGet := doRequest(mux, http.MethodGet, "/api/assets/"+strconvFormat(created.ID)+"/preview", nil)
	if wGet.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", wGet.Code, wGet.Body.String())
	}
	if wGet.Header().Get("Content-Type") != "image/png" {
		t.Errorf("Expected Content-Type image/png, got %s", wGet.Header().Get("Content-Type"))
	}
	if !bytes.Equal(wGet.Body.Bytes(), pngHeader) {
		t.Errorf("Retrieved image bytes mismatch")
	}
}

// 26. Retrieve Missing Preview -> 404
func TestGetPreview_Missing(t *testing.T) {
	mux, _, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Asset without preview"})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	wGet := doRequest(mux, http.MethodGet, "/api/assets/"+strconvFormat(created.ID)+"/preview", nil)
	if wGet.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found, got %d", wGet.Code)
	}
}

// 27. Replace Existing Preview (PNG -> WebP cleans up PNG)
func TestUploadPreview_Replace(t *testing.T) {
	mux, previewsDir, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Asset to Replace"})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	pngHeader := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82")
	_ = doMultipartUpload(mux, "/api/assets/"+strconvFormat(created.ID)+"/preview", "file", "old.png", pngHeader)

	pngPath := filepath.Join(previewsDir, strconvFormat(created.ID)+".png")
	if _, err := os.Stat(pngPath); err != nil {
		t.Fatalf("Expected old png to exist before replace: %v", err)
	}

	webpHeader := []byte("RIFF\x1a\x00\x00\x00WEBPVP8 \x0e\x00\x00\x000\x01\x00\x9d\x01*\x01\x00\x01\x00\x00")
	wReplace := doMultipartUpload(mux, "/api/assets/"+strconvFormat(created.ID)+"/preview", "file", "new.webp", webpHeader)
	if wReplace.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on replace, got %d: %s", wReplace.Code, wReplace.Body.String())
	}

	// Verify old png is removed
	if _, err := os.Stat(pngPath); !os.IsNotExist(err) {
		t.Errorf("Expected old png file to be deleted, but it still exists")
	}

	// Verify new webp exists
	webpPath := filepath.Join(previewsDir, strconvFormat(created.ID)+".webp")
	if _, err := os.Stat(webpPath); err != nil {
		t.Errorf("Expected new webp file to exist: %v", err)
	}
}

// 28. Delete Preview - Success
func TestDeletePreview_Success(t *testing.T) {
	mux, previewsDir, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Asset to Delete Preview"})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	jpegHeader := []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00\xFF\xDB\x00C\x00data")
	_ = doMultipartUpload(mux, "/api/assets/"+strconvFormat(created.ID)+"/preview", "file", "img.jpg", jpegHeader)

	filePath := filepath.Join(previewsDir, strconvFormat(created.ID)+".jpg")
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("Expected preview file to exist: %v", err)
	}

	wDel := doRequest(mux, http.MethodDelete, "/api/assets/"+strconvFormat(created.ID)+"/preview", nil)
	if wDel.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on preview delete, got %d: %s", wDel.Code, wDel.Body.String())
	}

	// Verify file is gone from disk
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("Expected file to be removed from disk, but it still exists")
	}

	// Verify database preview_path is cleared
	wGetAsset := doRequest(mux, http.MethodGet, "/api/assets/"+strconvFormat(created.ID), nil)
	var reloaded asset.Asset
	_ = json.NewDecoder(wGetAsset.Body).Decode(&reloaded)
	if reloaded.PreviewPath != "" {
		t.Errorf("Expected empty preview_path in db, got %q", reloaded.PreviewPath)
	}
}

// 29. Delete Preview - Idempotent
func TestDeletePreview_Idempotent(t *testing.T) {
	mux, _, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Asset No Preview"})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	wDel := doRequest(mux, http.MethodDelete, "/api/assets/"+strconvFormat(created.ID)+"/preview", nil)
	if wDel.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for idempotent preview delete, got %d: %s", wDel.Code, wDel.Body.String())
	}
}

// 30. Asset Deletion Cleans Up Preview File
func TestAssetDelete_CleansUpPreview(t *testing.T) {
	mux, previewsDir, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Asset to fully delete"})
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	jpegHeader := []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00\xFF\xDB\x00C\x00data")
	_ = doMultipartUpload(mux, "/api/assets/"+strconvFormat(created.ID)+"/preview", "file", "preview.jpg", jpegHeader)

	filePath := filepath.Join(previewsDir, strconvFormat(created.ID)+".jpg")
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("Expected preview to exist before asset deletion: %v", err)
	}

	wDelAsset := doRequest(mux, http.MethodDelete, "/api/assets/"+strconvFormat(created.ID), nil)
	if wDelAsset.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on asset delete, got %d: %s", wDelAsset.Code, wDelAsset.Body.String())
	}

	// Verify file is removed
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("Expected preview file to be cleaned up after asset deletion, but it still exists")
	}
}

// 31. Delete Asset - Safety: Does NOT Delete or Modify Original Local Path
func TestDeleteAsset_DoesNotDeleteLocalPath(t *testing.T) {
	mux, previewsDir, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	// Create a simulated local VRChat asset directory with files outside the database/previews
	localAssetDir, err := os.MkdirTemp("", "vrchat-user-asset-*")
	if err != nil {
		t.Fatalf("Failed to create temp local asset dir: %v", err)
	}
	defer os.RemoveAll(localAssetDir)

	subDir := filepath.Join(localAssetDir, "Textures")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subDir: %v", err)
	}

	modelFile := filepath.Join(localAssetDir, "avatar.fbx")
	modelContent := []byte("binary FBX 3D model data")
	if err := os.WriteFile(modelFile, modelContent, 0644); err != nil {
		t.Fatalf("Failed to write model file: %v", err)
	}

	textureFile := filepath.Join(subDir, "hair.png")
	textureContent := []byte("PNG texture data")
	if err := os.WriteFile(textureFile, textureContent, 0644); err != nil {
		t.Fatalf("Failed to write texture file: %v", err)
	}

	// Create asset pointing to this local_path
	wCreate := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:      "VRChat Hair Asset",
		LocalPath: localAssetDir,
	})
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("Failed to create asset: %s", wCreate.Body.String())
	}
	var created asset.Asset
	_ = json.NewDecoder(wCreate.Body).Decode(&created)

	// Also upload an application-owned preview image to verify preview cleanup works alongside local path preservation
	jpegHeader := []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00\xFF\xDB\x00C\x00data")
	_ = doMultipartUpload(mux, "/api/assets/"+strconvFormat(created.ID)+"/preview", "file", "preview.jpg", jpegHeader)

	previewFilePath := filepath.Join(previewsDir, strconvFormat(created.ID)+".jpg")
	if _, err := os.Stat(previewFilePath); err != nil {
		t.Fatalf("Expected preview to exist before asset deletion: %v", err)
	}

	// Now delete the asset record
	wDel := doRequest(mux, http.MethodDelete, "/api/assets/"+strconvFormat(created.ID), nil)
	if wDel.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on asset delete, got %d: %s", wDel.Code, wDel.Body.String())
	}

	// 1. Verify asset record is gone from DB
	wGet := doRequest(mux, http.MethodGet, "/api/assets/"+strconvFormat(created.ID), nil)
	if wGet.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found after deletion, got %d", wGet.Code)
	}

	// 2. Verify application-owned preview file WAS deleted
	if _, err := os.Stat(previewFilePath); !os.IsNotExist(err) {
		t.Errorf("Expected preview file under previewsDir to be deleted, but it still exists")
	}

	// 3. CRITICAL SAFETY CHECK: Verify local_path directory STILL EXISTS
	if fi, err := os.Stat(localAssetDir); err != nil || !fi.IsDir() {
		t.Fatalf("FATAL SAFETY VIOLATION: Local asset directory was deleted or modified! err=%v", err)
	}

	// 4. Verify subdirectories and files within local_path STILL EXIST and are UNMODIFIED
	readModel, err := os.ReadFile(modelFile)
	if err != nil {
		t.Fatalf("FATAL SAFETY VIOLATION: Model file in local_path was deleted! err=%v", err)
	}
	if !bytes.Equal(readModel, modelContent) {
		t.Fatalf("FATAL SAFETY VIOLATION: Model file in local_path was modified!")
	}

	readTexture, err := os.ReadFile(textureFile)
	if err != nil {
		t.Fatalf("FATAL SAFETY VIOLATION: Texture file in local_path was deleted! err=%v", err)
	}
	if !bytes.Equal(readTexture, textureContent) {
		t.Fatalf("FATAL SAFETY VIOLATION: Texture file in local_path was modified!")
	}
}

// 32. Tag API - GetTags and CreateTag
func TestTags_API(t *testing.T) {
	mux, _, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	// Initial tags should be empty
	wGet := doRequest(mux, http.MethodGet, "/api/tags", nil)
	if wGet.Code != http.StatusOK {
		t.Fatalf("Expected 200 for GET /api/tags, got %d", wGet.Code)
	}
	var tags []asset.Tag
	_ = json.NewDecoder(wGet.Body).Decode(&tags)
	if len(tags) != 0 {
		t.Errorf("Expected 0 tags initially, got %d", len(tags))
	}

	// Create a tag
	wCreateTag := doRequest(mux, http.MethodPost, "/api/tags", asset.CreateTagRequest{Name: "Goth"})
	if wCreateTag.Code != http.StatusCreated {
		t.Fatalf("Expected 201 for POST /api/tags, got %d: %s", wCreateTag.Code, wCreateTag.Body.String())
	}
	var createdTag asset.Tag
	_ = json.NewDecoder(wCreateTag.Body).Decode(&createdTag)
	if createdTag.Name != "Goth" || createdTag.ID == 0 {
		t.Errorf("Unexpected created tag: %+v", createdTag)
	}

	// Create empty tag should return 400
	wEmpty := doRequest(mux, http.MethodPost, "/api/tags", asset.CreateTagRequest{Name: "   "})
	if wEmpty.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for empty tag name, got %d", wEmpty.Code)
	}

	// Creating duplicate tag returns existing tag
	wDup := doRequest(mux, http.MethodPost, "/api/tags", asset.CreateTagRequest{Name: "goth"})
	if wDup.Code != http.StatusCreated {
		t.Fatalf("Expected 201 for duplicate tag name, got %d", wDup.Code)
	}
	var dupTag asset.Tag
	_ = json.NewDecoder(wDup.Body).Decode(&dupTag)
	if dupTag.ID != createdTag.ID {
		t.Errorf("Expected same tag ID for case-insensitive duplicate, got %d vs %d", dupTag.ID, createdTag.ID)
	}

	// Now GET /api/tags should return the tag
	wGet2 := doRequest(mux, http.MethodGet, "/api/tags", nil)
	var tags2 []asset.Tag
	_ = json.NewDecoder(wGet2.Body).Decode(&tags2)
	if len(tags2) != 1 || tags2[0].Name != "Goth" {
		t.Errorf("Expected 1 tag named 'Goth', got %+v", tags2)
	}
}

func strconvFormat(id int64) string {
	return strconv.FormatInt(id, 10)
}

// 33. Milestone 7 - Favorite Toggle and Filtering
func TestFavorite_ToggleAndFilter(t *testing.T) {
	mux, _, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	// Create asset A and B
	wA := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name: "Favorite Asset A",
	})
	if wA.Code != http.StatusCreated {
		t.Fatalf("Failed to create asset A: %d", wA.Code)
	}
	var assetA asset.Asset
	_ = json.NewDecoder(wA.Body).Decode(&assetA)
	if assetA.IsFavorite {
		t.Errorf("Expected asset A to start with IsFavorite=false")
	}

	wB := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name: "Standard Asset B",
	})
	if wB.Code != http.StatusCreated {
		t.Fatalf("Failed to create asset B: %d", wB.Code)
	}
	var assetB asset.Asset
	_ = json.NewDecoder(wB.Body).Decode(&assetB)

	// Toggle A to favorite
	wToggle := doRequest(mux, http.MethodPost, "/api/assets/"+strconvFormat(assetA.ID)+"/favorite", nil)
	if wToggle.Code != http.StatusOK {
		t.Fatalf("Failed to toggle favorite: %d, %s", wToggle.Code, wToggle.Body.String())
	}
	var toggledA asset.Asset
	_ = json.NewDecoder(wToggle.Body).Decode(&toggledA)
	if !toggledA.IsFavorite {
		t.Errorf("Expected toggled asset A to have IsFavorite=true")
	}

	// Filter favorite=true
	wFavOnly := doRequest(mux, http.MethodGet, "/api/assets?favorite=true", nil)
	var favAssets []asset.Asset
	_ = json.NewDecoder(wFavOnly.Body).Decode(&favAssets)
	if len(favAssets) != 1 || favAssets[0].ID != assetA.ID {
		t.Errorf("Expected only asset A in favorite=true, got %d assets", len(favAssets))
	}

	// Filter favorite=false
	wNonFav := doRequest(mux, http.MethodGet, "/api/assets?favorite=false", nil)
	var nonFavAssets []asset.Asset
	_ = json.NewDecoder(wNonFav.Body).Decode(&nonFavAssets)
	if len(nonFavAssets) != 1 || nonFavAssets[0].ID != assetB.ID {
		t.Errorf("Expected only asset B in favorite=false, got %d assets", len(nonFavAssets))
	}

	// Explicitly toggle back to false with request body
	isFavFalse := false
	wToggleBack := doRequest(mux, http.MethodPost, "/api/assets/"+strconvFormat(assetA.ID)+"/favorite", asset.ToggleFavoriteRequest{
		IsFavorite: &isFavFalse,
	})
	if wToggleBack.Code != http.StatusOK {
		t.Fatalf("Failed to toggle favorite back: %d", wToggleBack.Code)
	}
	var untoggledA asset.Asset
	_ = json.NewDecoder(wToggleBack.Body).Decode(&untoggledA)
	if untoggledA.IsFavorite {
		t.Errorf("Expected asset A to have IsFavorite=false after untoggle")
	}
}

// 34. Milestone 7 - Search Across Name, Author, and Tags
func TestSearch_AcrossNameAuthorTags(t *testing.T) {
	mux, _, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	_ = doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:   "Shinra Outfit",
		Author: "Torikago Store",
		Tags:   []string{"Gothic", "Physbone"},
	})
	_ = doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:   "Kikyo Casual",
		Author: "Chibiko Studio",
		Tags:   []string{"TorikagoVibe", "Casual"},
	})

	// Search by name (case-insensitive)
	wName := doRequest(mux, http.MethodGet, "/api/assets?search=shinra", nil)
	var resName []asset.Asset
	_ = json.NewDecoder(wName.Body).Decode(&resName)
	if len(resName) != 1 || resName[0].Name != "Shinra Outfit" {
		t.Errorf("Search by name failed: got %d assets", len(resName))
	}

	// Search by author
	wAuthor := doRequest(mux, http.MethodGet, "/api/assets?search=chibiko", nil)
	var resAuthor []asset.Asset
	_ = json.NewDecoder(wAuthor.Body).Decode(&resAuthor)
	if len(resAuthor) != 1 || resAuthor[0].Name != "Kikyo Casual" {
		t.Errorf("Search by author failed: got %d assets", len(resAuthor))
	}

	// Search by tag
	wTag := doRequest(mux, http.MethodGet, "/api/assets?search=physbone", nil)
	var resTag []asset.Asset
	_ = json.NewDecoder(wTag.Body).Decode(&resTag)
	if len(resTag) != 1 || resTag[0].Name != "Shinra Outfit" {
		t.Errorf("Search by tag failed: got %d assets", len(resTag))
	}

	// Search term matching author in asset 1 and tag in asset 2
	wShared := doRequest(mux, http.MethodGet, "/api/assets?search=torikago", nil)
	var resShared []asset.Asset
	_ = json.NewDecoder(wShared.Body).Decode(&resShared)
	if len(resShared) != 2 {
		t.Errorf("Search matching author and tag expected 2 results, got %d", len(resShared))
	}
}

// 35. Milestone 7 - Multi-Tag Filtering with AND semantics
func TestMultiTag_Filter(t *testing.T) {
	mux, _, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	_ = doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name: "Asset 1",
		Tags: []string{"Anime", "Female", "Quest"},
	})
	_ = doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name: "Asset 2",
		Tags: []string{"Anime", "Female", "PC"},
	})
	_ = doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name: "Asset 3",
		Tags: []string{"Anime", "Male"},
	})

	// Filter with comma-separated tags Anime,Female (AND)
	w2 := doRequest(mux, http.MethodGet, "/api/assets?tags=Anime,Female", nil)
	var res2 []asset.Asset
	_ = json.NewDecoder(w2.Body).Decode(&res2)
	if len(res2) != 2 {
		t.Errorf("Expected 2 assets with Anime AND Female, got %d", len(res2))
	}

	// Filter with multiple tags parameters: tags=Anime&tags=Female&tags=Quest
	w3 := doRequest(mux, http.MethodGet, "/api/assets?tags=Anime&tags=Female&tags=Quest", nil)
	var res3 []asset.Asset
	_ = json.NewDecoder(w3.Body).Decode(&res3)
	if len(res3) != 1 || res3[0].Name != "Asset 1" {
		t.Errorf("Expected 1 asset with Anime, Female, Quest, got %d", len(res3))
	}

	// Non-matching tag
	wNone := doRequest(mux, http.MethodGet, "/api/assets?tags=NonExistentTag", nil)
	var resNone []asset.Asset
	_ = json.NewDecoder(wNone.Body).Decode(&resNone)
	if len(resNone) != 0 {
		t.Errorf("Expected 0 assets for non-existent tag, got %d", len(resNone))
	}
}

// 36. Milestone 7 - Sorting Options
func TestSorting_Options(t *testing.T) {
	mux, _, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	_ = doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Charlie"})
	_ = doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Alpha"})
	_ = doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{Name: "Bravo"})

	// Sort name_asc
	wAsc := doRequest(mux, http.MethodGet, "/api/assets?sort=name_asc", nil)
	var resAsc []asset.Asset
	_ = json.NewDecoder(wAsc.Body).Decode(&resAsc)
	if len(resAsc) != 3 || resAsc[0].Name != "Alpha" || resAsc[1].Name != "Bravo" || resAsc[2].Name != "Charlie" {
		t.Errorf("name_asc sort incorrect: %+v", resAsc)
	}

	// Sort name_desc
	wDesc := doRequest(mux, http.MethodGet, "/api/assets?sort=name_desc", nil)
	var resDesc []asset.Asset
	_ = json.NewDecoder(wDesc.Body).Decode(&resDesc)
	if len(resDesc) != 3 || resDesc[0].Name != "Charlie" || resDesc[1].Name != "Bravo" || resDesc[2].Name != "Alpha" {
		t.Errorf("name_desc sort incorrect: %+v", resDesc)
	}
}

// 37. Milestone 7 - Filter by Preview, Booth URL, and Local Status
func TestFilter_PreviewBoothLocalStatus(t *testing.T) {
	mux, previewsDir, cleanup := setupTestServerWithConfig(t, nil)
	defer cleanup()

	// Create a temp file on disk for existing local_path test
	tempFile, err := os.CreateTemp("", "vram-test-file-*.unitypackage")
	if err != nil {
		t.Fatalf("failed to create temp test file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempPath := tempFile.Name()
	_ = tempFile.Close()

	// Asset 1: has booth_url and valid local_path
	w1 := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:      "Asset Complete",
		BoothURL:  "https://booth.pm/en/items/12345",
		LocalPath: tempPath,
	})
	var a1 asset.Asset
	_ = json.NewDecoder(w1.Body).Decode(&a1)

	// Asset 2: missing local path and no booth URL
	w2 := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name:      "Asset Missing Path",
		LocalPath: "C:\\non\\existent\\path\\model.fbx",
	})
	var a2 asset.Asset
	_ = json.NewDecoder(w2.Body).Decode(&a2)

	// Asset 3: no local path specified
	w3 := doRequest(mux, http.MethodPost, "/api/assets", asset.CreateAssetRequest{
		Name: "Asset Empty Path",
	})
	var a3 asset.Asset
	_ = json.NewDecoder(w3.Body).Decode(&a3)

	// Filter has_booth=true
	wBooth := doRequest(mux, http.MethodGet, "/api/assets?has_booth=true", nil)
	var resBooth []asset.Asset
	_ = json.NewDecoder(wBooth.Body).Decode(&resBooth)
	if len(resBooth) != 1 || resBooth[0].ID != a1.ID {
		t.Errorf("has_booth=true filter failed, got %d assets", len(resBooth))
	}

	// Filter local_status=available
	wAvail := doRequest(mux, http.MethodGet, "/api/assets?local_status=available", nil)
	var resAvail []asset.Asset
	_ = json.NewDecoder(wAvail.Body).Decode(&resAvail)
	if len(resAvail) != 1 || resAvail[0].ID != a1.ID {
		t.Errorf("local_status=available failed, got %d assets", len(resAvail))
	}

	// Filter local_status=missing
	wMiss := doRequest(mux, http.MethodGet, "/api/assets?local_status=missing", nil)
	var resMiss []asset.Asset
	_ = json.NewDecoder(wMiss.Body).Decode(&resMiss)
	if len(resMiss) != 1 || resMiss[0].ID != a2.ID {
		t.Errorf("local_status=missing failed, got %d assets", len(resMiss))
	}

	// Filter local_status=not_specified
	wNotSpec := doRequest(mux, http.MethodGet, "/api/assets?local_status=not_specified", nil)
	var resNotSpec []asset.Asset
	_ = json.NewDecoder(wNotSpec.Body).Decode(&resNotSpec)
	if len(resNotSpec) != 1 || resNotSpec[0].ID != a3.ID {
		t.Errorf("local_status=not_specified failed, got %d assets", len(resNotSpec))
	}

	// Test BatchStatus API
	wBatch := doRequest(mux, http.MethodPost, "/api/assets/batch-status", asset.BatchStatusRequest{
		IDs: []int64{a1.ID, a2.ID, a3.ID},
	})
	if wBatch.Code != http.StatusOK {
		t.Fatalf("Batch status returned %d", wBatch.Code)
	}
	var batchRes asset.BatchStatusResponse
	_ = json.NewDecoder(wBatch.Body).Decode(&batchRes)
	if batchRes.Statuses[strconvFormat(a1.ID)] != true {
		t.Errorf("Expected asset 1 to exist in batch status")
	}
	if batchRes.Statuses[strconvFormat(a2.ID)] != false {
		t.Errorf("Expected asset 2 to NOT exist in batch status")
	}
	if batchRes.Statuses[strconvFormat(a3.ID)] != false {
		t.Errorf("Expected asset 3 to NOT exist in batch status")
	}

	_ = previewsDir
}
