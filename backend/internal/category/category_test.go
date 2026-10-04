package category_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"vrchat-asset-manager/backend/internal/category"
	"vrchat-asset-manager/backend/internal/database"
	"vrchat-asset-manager/backend/migrations"
)

func setup(t *testing.T) (*http.ServeMux, *database.DB) {
	t.Helper()
	db, err := database.Connect(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mux := http.NewServeMux()
	category.NewHandler(category.NewRepository(db.DB)).RegisterRoutes(mux)
	return mux, db
}

func do(mux *http.ServeMux, method, target string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(method, target, &buf))
	return w
}

func idOf(t *testing.T, db *database.DB, name string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow("SELECT id FROM categories WHERE name = ?", name).Scan(&id); err != nil {
		t.Fatalf("category %q not found: %v", name, err)
	}
	return id
}

func path(id int64) string { return "/api/categories/" + strconv.FormatInt(id, 10) }

func TestCreateChildAppearsUnderParent(t *testing.T) {
	mux, db := setup(t)
	faceID := idOf(t, db, "Face")

	w := do(mux, http.MethodPost, "/api/categories", map[string]any{"name": "Nail", "parent_id": faceID})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created category.Category
	_ = json.NewDecoder(w.Body).Decode(&created)
	if created.ParentID == nil || *created.ParentID != faceID || created.SortOrder != 40 {
		t.Errorf("unexpected created category: %+v", created)
	}

	var list []category.Category
	_ = json.NewDecoder(do(mux, http.MethodGet, "/api/categories", nil).Body).Decode(&list)
	for i, c := range list {
		if c.Name == "Makeup" {
			if i+1 >= len(list) || list[i+1].Name != "Nail" {
				t.Errorf("expected Nail right after Makeup in display order")
			}
		}
	}
}

func TestValidationRules(t *testing.T) {
	mux, db := setup(t)
	faceID := idOf(t, db, "Face")
	eyesID := idOf(t, db, "Eyes")
	hairID := idOf(t, db, "Hair")

	cases := []struct {
		name   string
		method string
		target string
		body   map[string]any
		want   int
	}{
		{"empty name", http.MethodPost, "/api/categories", map[string]any{"name": " "}, http.StatusBadRequest},
		{"duplicate name (case-insensitive)", http.MethodPost, "/api/categories", map[string]any{"name": "hair"}, http.StatusConflict},
		{"grandchild not allowed", http.MethodPost, "/api/categories", map[string]any{"name": "Iris", "parent_id": eyesID}, http.StatusBadRequest},
		{"missing parent", http.MethodPost, "/api/categories", map[string]any{"name": "X", "parent_id": 9999}, http.StatusBadRequest},
		{"parent with children cannot become a child", http.MethodPut, path(faceID), map[string]any{"name": "Face", "parent_id": hairID}, http.StatusBadRequest},
		{"own parent", http.MethodPut, path(hairID), map[string]any{"name": "Hair", "parent_id": hairID}, http.StatusBadRequest},
		{"delete category with children", http.MethodDelete, path(faceID), nil, http.StatusBadRequest},
		{"unknown id", http.MethodPut, "/api/categories/9999", map[string]any{"name": "Y"}, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body any
			if tc.body != nil {
				body = tc.body
			}
			if w := do(mux, tc.method, tc.target, body); w.Code != tc.want {
				t.Errorf("expected %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestRenameKeepsSortOrderAndDeleteUncategorizesAssets(t *testing.T) {
	mux, db := setup(t)
	hairID := idOf(t, db, "Hair")

	w := do(mux, http.MethodPut, path(hairID), map[string]any{"name": "Hairstyle"})
	var renamed category.Category
	_ = json.NewDecoder(w.Body).Decode(&renamed)
	if w.Code != http.StatusOK || renamed.Name != "Hairstyle" || renamed.SortOrder != 30 {
		t.Fatalf("unexpected rename result %d: %+v", w.Code, renamed)
	}

	res, _ := db.Exec("INSERT INTO assets (name, category_id) VALUES ('Twin Tail', ?)", hairID)
	assetID, _ := res.LastInsertId()

	if w := do(mux, http.MethodDelete, path(hairID), nil); w.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d: %s", w.Code, w.Body.String())
	}
	var catID *int64
	_ = db.QueryRow("SELECT category_id FROM assets WHERE id = ?", assetID).Scan(&catID)
	if catID != nil {
		t.Errorf("expected asset to become uncategorized, got category %d", *catID)
	}
}
