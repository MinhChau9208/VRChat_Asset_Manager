package asset_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"vrchat-asset-manager/backend/internal/asset"
	"vrchat-asset-manager/backend/internal/category"
)

func createAsset(t *testing.T, mux *http.ServeMux, req any) asset.Asset {
	t.Helper()
	w := doRequest(mux, http.MethodPost, "/api/assets", req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed %d: %s", w.Code, w.Body.String())
	}
	var a asset.Asset
	_ = json.NewDecoder(w.Body).Decode(&a)
	return a
}

func getAsset(t *testing.T, mux *http.ServeMux, id int64) asset.Asset {
	t.Helper()
	w := doRequest(mux, http.MethodGet, "/api/assets/"+strconvFormat(id), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get failed %d: %s", w.Code, w.Body.String())
	}
	var a asset.Asset
	_ = json.NewDecoder(w.Body).Decode(&a)
	return a
}

func listNames(t *testing.T, mux *http.ServeMux, query string) []string {
	t.Helper()
	w := doRequest(mux, http.MethodGet, "/api/assets"+query, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list failed %d: %s", w.Code, w.Body.String())
	}
	var assets []asset.Asset
	_ = json.NewDecoder(w.Body).Decode(&assets)
	names := []string{}
	for _, a := range assets {
		names = append(names, a.Name)
	}
	return names
}

func TestDraftsHiddenByDefault(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	createAsset(t, mux, map[string]any{"name": "Confirmed"})
	draft := createAsset(t, mux, map[string]any{"name": "Scanned", "status": "draft"})
	if draft.Status != "draft" {
		t.Fatalf("expected draft status, got %q", draft.Status)
	}

	if got := listNames(t, mux, ""); len(got) != 1 || got[0] != "Confirmed" {
		t.Errorf("default list should hide drafts, got %v", got)
	}
	if got := listNames(t, mux, "?status=draft"); len(got) != 1 || got[0] != "Scanned" {
		t.Errorf("status=draft should list drafts only, got %v", got)
	}
	if got := listNames(t, mux, "?status=all"); len(got) != 2 {
		t.Errorf("status=all should list everything, got %v", got)
	}

	var stats asset.LibraryStats
	_ = json.NewDecoder(doRequest(mux, http.MethodGet, "/api/stats", nil).Body).Decode(&stats)
	if stats.Total != 1 {
		t.Errorf("stats should ignore drafts, got total=%d", stats.Total)
	}

	// Confirming a draft makes it visible.
	w := doRequest(mux, http.MethodPut, "/api/assets/"+strconvFormat(draft.ID), map[string]any{"name": "Scanned", "status": "active"})
	if w.Code != http.StatusOK {
		t.Fatalf("confirm failed %d: %s", w.Code, w.Body.String())
	}
	if got := listNames(t, mux, ""); len(got) != 2 {
		t.Errorf("confirmed draft should be listed, got %v", got)
	}

	if w := doRequest(mux, http.MethodPost, "/api/assets", map[string]any{"name": "X", "status": "archived"}); w.Code != http.StatusBadRequest {
		t.Errorf("invalid status should be rejected, got %d", w.Code)
	}
}

func TestParentCategoryFilterIncludesChildren(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	var categories []category.Category
	_ = json.NewDecoder(doRequest(mux, http.MethodGet, "/api/categories", nil).Body).Decode(&categories)
	ids := map[string]int64{}
	for _, c := range categories {
		ids[c.Name] = c.ID
	}
	clothesID, outfitID, hairID := ids["Clothes"], ids["Outfit"], ids["Hair"]

	createAsset(t, mux, map[string]any{"name": "Gothic Doll", "category_id": clothesID})
	createAsset(t, mux, map[string]any{"name": "Outfit Set", "category_id": outfitID})
	createAsset(t, mux, map[string]any{"name": "Twin Tail", "category_id": hairID})

	if got := listNames(t, mux, "?category=Outfit"); len(got) != 2 {
		t.Errorf("Outfit should include its own and Clothes assets, got %v", got)
	}
	if got := listNames(t, mux, "?category="+strconvFormat(outfitID)); len(got) != 2 {
		t.Errorf("filter by parent id should include children, got %v", got)
	}
	if got := listNames(t, mux, "?category=Clothes"); len(got) != 1 || got[0] != "Gothic Doll" {
		t.Errorf("child filter should not include parent assets, got %v", got)
	}
}

func TestAssetFiles(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	dir := t.TempDir()
	v111 := filepath.Join(dir, "Kipfel_1.1.1")
	v120 := filepath.Join(dir, "Kipfel_1.2.0")
	zip := filepath.Join(dir, "Kipfel_1.2.0.zip")
	_ = os.Mkdir(v120, 0755)
	_ = os.WriteFile(zip, []byte("PK"), 0644)

	a := createAsset(t, mux, map[string]any{"name": "Kipfel", "local_path": v111})
	if len(a.Files) != 1 || a.Files[0].Path != v111 || a.Files[0].Kind != "folder" || a.Files[0].Exists {
		t.Fatalf("local_path should become the first (missing) file, got %+v", a.Files)
	}

	// Changing local_path renames the primary file instead of adding a new one.
	w := doRequest(mux, http.MethodPut, "/api/assets/"+strconvFormat(a.ID), map[string]any{"name": "Kipfel", "local_path": v120})
	if w.Code != http.StatusOK {
		t.Fatalf("update failed %d: %s", w.Code, w.Body.String())
	}
	a = getAsset(t, mux, a.ID)
	if len(a.Files) != 1 || a.Files[0].Path != v120 || !a.Files[0].Exists {
		t.Fatalf("primary file should follow local_path, got %+v", a.Files)
	}

	w = doRequest(mux, http.MethodPost, "/api/assets/"+strconvFormat(a.ID)+"/files", map[string]any{"path": zip, "version": "1.2.0"})
	if w.Code != http.StatusCreated {
		t.Fatalf("add file failed %d: %s", w.Code, w.Body.String())
	}
	var added asset.AssetFile
	_ = json.NewDecoder(w.Body).Decode(&added)
	if added.Kind != "archive" || added.Version != "1.2.0" || !added.Exists {
		t.Errorf("unexpected added file: %+v", added)
	}

	other := createAsset(t, mux, map[string]any{"name": "Other"})
	if w := doRequest(mux, http.MethodPost, "/api/assets/"+strconvFormat(other.ID)+"/files", map[string]any{"path": zip}); w.Code != http.StatusConflict {
		t.Errorf("a path can belong to one asset only, got %d", w.Code)
	}

	// Unlinking the primary file promotes the remaining one.
	a = getAsset(t, mux, a.ID)
	var primaryID int64
	for _, f := range a.Files {
		if f.Path == v120 {
			primaryID = f.ID
		}
	}
	if w := doRequest(mux, http.MethodDelete, "/api/assets/"+strconvFormat(a.ID)+"/files/"+strconvFormat(primaryID), nil); w.Code != http.StatusOK {
		t.Fatalf("delete file failed %d: %s", w.Code, w.Body.String())
	}
	a = getAsset(t, mux, a.ID)
	if a.LocalPath != zip || len(a.Files) != 1 {
		t.Errorf("expected zip to become primary, got local_path=%q files=%+v", a.LocalPath, a.Files)
	}
	if _, err := os.Stat(v120); err != nil {
		t.Errorf("unlinking must never touch files on disk: %v", err)
	}
}

func TestCompatibleAvatars(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	kipfel := createAsset(t, mux, map[string]any{"name": "Kipfel"})
	outfit := createAsset(t, mux, map[string]any{
		"name": "Small Lady",
		"compatible_avatars": []map[string]any{
			{"avatar_asset_id": kipfel.ID},
			{"avatar_name": "Manuka"},
			{"avatar_name": "manuka"}, // duplicate by name
		},
	})
	createAsset(t, mux, map[string]any{"name": "Unrelated"})

	if len(outfit.CompatibleAvatars) != 2 {
		t.Fatalf("expected 2 compatible avatars, got %+v", outfit.CompatibleAvatars)
	}
	if got := listNames(t, mux, "?compatible_with="+strconvFormat(kipfel.ID)); len(got) != 1 || got[0] != "Small Lady" {
		t.Errorf("compatible_with should list the outfit only, got %v", got)
	}

	// Renaming the avatar updates the label.
	doRequest(mux, http.MethodPut, "/api/assets/"+strconvFormat(kipfel.ID), map[string]any{"name": "Kipfel v1.2"})
	outfit = getAsset(t, mux, outfit.ID)
	found := false
	for _, c := range outfit.CompatibleAvatars {
		if c.AvatarAssetID != nil && *c.AvatarAssetID == kipfel.ID && c.AvatarName == "Kipfel v1.2" {
			found = true
		}
	}
	if !found {
		t.Errorf("avatar rename should propagate, got %+v", outfit.CompatibleAvatars)
	}

	// Updating without compatible_avatars keeps them; an empty list clears them.
	doRequest(mux, http.MethodPut, "/api/assets/"+strconvFormat(outfit.ID), map[string]any{"name": "Small Lady"})
	if got := getAsset(t, mux, outfit.ID).CompatibleAvatars; len(got) != 2 {
		t.Errorf("omitting compatible_avatars should keep them, got %+v", got)
	}

	// Deleting the avatar keeps the name but drops the link.
	doRequest(mux, http.MethodDelete, "/api/assets/"+strconvFormat(kipfel.ID), nil)
	for _, c := range getAsset(t, mux, outfit.ID).CompatibleAvatars {
		if c.AvatarName == "Kipfel v1.2" && c.AvatarAssetID != nil {
			t.Errorf("deleted avatar link should be cleared, got %+v", c)
		}
	}

	doRequest(mux, http.MethodPut, "/api/assets/"+strconvFormat(outfit.ID), map[string]any{"name": "Small Lady", "compatible_avatars": []any{}})
	if got := getAsset(t, mux, outfit.ID).CompatibleAvatars; len(got) != 0 {
		t.Errorf("empty list should clear compatibility, got %+v", got)
	}

	if w := doRequest(mux, http.MethodPost, "/api/assets", map[string]any{
		"name": "Bad", "compatible_avatars": []map[string]any{{"avatar_asset_id": 9999}},
	}); w.Code != http.StatusBadRequest {
		t.Errorf("unknown avatar asset should be rejected, got %d", w.Code)
	}
}

func TestBulkUpdate(t *testing.T) {
	mux, cleanup := setupTestServer(t)
	defer cleanup()

	kipfel := createAsset(t, mux, map[string]any{"name": "Kipfel"})
	a := createAsset(t, mux, map[string]any{"name": "Dress", "tags": []string{"cute"}, "category_id": 3})
	b := createAsset(t, mux, map[string]any{"name": "Hat", "compatible_avatars": []map[string]any{{"avatar_name": "Manuka"}}})

	w := doRequest(mux, http.MethodPost, "/api/assets/bulk", map[string]any{
		"asset_ids":              []int64{a.ID, b.ID},
		"set_category":           true,
		"category_id":            5,
		"add_tags":               []string{"Kipfel", "cute"},
		"add_compatible_avatars": []map[string]any{{"avatar_asset_id": kipfel.ID}},
		"is_favorite":            true,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("bulk failed %d: %s", w.Code, w.Body.String())
	}

	ga, gb := getAsset(t, mux, a.ID), getAsset(t, mux, b.ID)
	for _, x := range []asset.Asset{ga, gb} {
		if x.CategoryID == nil || *x.CategoryID != 5 || !x.IsFavorite {
			t.Errorf("%s: category/favorite not applied: %+v", x.Name, x)
		}
	}
	if len(ga.Tags) != 2 || len(gb.Tags) != 2 {
		t.Errorf("tags should be added without duplicates: %v / %v", ga.Tags, gb.Tags)
	}
	if len(gb.CompatibleAvatars) != 2 {
		t.Errorf("compatibility should be added, keeping Manuka: %+v", gb.CompatibleAvatars)
	}
	if ga.Name != "Dress" {
		t.Errorf("bulk must not touch other fields, got name %q", ga.Name)
	}

	// Only the fields that are set change: no category change here.
	doRequest(mux, http.MethodPost, "/api/assets/bulk", map[string]any{"asset_ids": []int64{a.ID}, "add_tags": []string{"new"}})
	if got := getAsset(t, mux, a.ID); got.CategoryID == nil || *got.CategoryID != 5 {
		t.Errorf("category should be untouched without set_category")
	}

	// Clearing the category.
	doRequest(mux, http.MethodPost, "/api/assets/bulk", map[string]any{"asset_ids": []int64{a.ID}, "set_category": true, "category_id": nil})
	if got := getAsset(t, mux, a.ID); got.CategoryID != nil {
		t.Errorf("category should be cleared")
	}

	for name, body := range map[string]map[string]any{
		"empty ids":        {"asset_ids": []int64{}},
		"unknown category": {"asset_ids": []int64{a.ID}, "set_category": true, "category_id": 9999},
	} {
		if w := doRequest(mux, http.MethodPost, "/api/assets/bulk", body); w.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", name, w.Code)
		}
	}
	if w := doRequest(mux, http.MethodPost, "/api/assets/bulk", map[string]any{"asset_ids": []int64{9999}, "add_tags": []string{"x"}}); w.Code != http.StatusNotFound {
		t.Errorf("unknown asset: expected 404, got %d", w.Code)
	}
}
