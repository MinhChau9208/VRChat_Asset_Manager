package scanner_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"vrchat-asset-manager/backend/internal/asset"
	"vrchat-asset-manager/backend/internal/database"
	"vrchat-asset-manager/backend/internal/scanner"
	"vrchat-asset-manager/backend/migrations"
)

// A tiny valid PNG header is enough for content sniffing.
var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

type env struct {
	mux      *http.ServeMux
	db       *database.DB
	lib      string
	previews string
}

func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

// setup builds a miniature copy of the real library layout.
func setup(t *testing.T) *env {
	t.Helper()
	tmp := t.TempDir()
	lib := filepath.Join(tmp, "Unity Materials")
	j := func(parts ...string) string { return filepath.Join(append([]string{lib}, parts...)...) }

	write(t, j("Models", "Kipfel_1.2.0", "Kipfel_1.2.0.unitypackage"), []byte("x"))
	write(t, j("Models", "Zips", "Kipfel v1.1.1.zip"), []byte("PK"))
	write(t, j("Models", "Meccha_Avatar", "Meccha.unitypackage"), []byte("x"))
	write(t, j("Models", "Meccha_Avatar", "main.png"), pngBytes)

	write(t, j("Clothes", "hamanosis_Small_Lady_Kipfel", "Small.unitypackage"), []byte("x"))
	write(t, j("Clothes", "hamanosis_Small_Lady_Kipfel", "ReadMe", "Small Lady - BOOTH.url"),
		[]byte("[InternetShortcut]\nURL=https://hamanosis.booth.pm/items/6834468\n"))
	write(t, j("Clothes", "hamanosis_Small_Lady_Kipfel", "ReadMe", "ReadMe.txt"),
		[]byte("対応アバター https://mukumi.booth.pm/items/5813187\nlilToon https://booth.pm/ja/items/3087170\n"))
	write(t, j("Clothes", "LEGACY", "hamanosis_Small_Lady_Kipfel.zip"), []byte("PK"))
	write(t, j("Clothes", "LEGACY", "BukiyouTwinTail_1.1.0.zip"), []byte("PK"))
	write(t, j("Hair", "BukiyouTwinTail_1.1.0", "Hair.unitypackage"), []byte("x"))

	write(t, j("PianoGimick", "4460917 avatargimmick 8ya_Undersea Piano 2", "readme.txt"), []byte("hi"))
	write(t, j("Audio", "song.mp3"), []byte("ID3"))
	write(t, j("AvatarPass", "secret.txt"), []byte("password https://booth.pm/ja/items/9999999"))

	db, err := database.Connect(filepath.Join(tmp, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(migrations.FS); err != nil {
		t.Fatal(err)
	}

	previews := filepath.Join(tmp, "previews")
	repo := asset.NewRepository(db.DB)
	mux := http.NewServeMux()
	asset.NewHandler(repo).RegisterRoutes(mux)
	scanner.NewHandler(scanner.NewService(db.DB, repo, previews)).RegisterRoutes(mux)
	return &env{mux: mux, db: db, lib: lib, previews: previews}
}

func (e *env) do(t *testing.T, method, target string, body any, out any) int {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, httptest.NewRequest(method, target, &buf))
	if out != nil {
		_ = json.NewDecoder(w.Body).Decode(out)
	}
	return w.Code
}

func (e *env) categoryID(t *testing.T, name string) int64 {
	var id int64
	if err := e.db.QueryRow("SELECT id FROM categories WHERE name = ?", name).Scan(&id); err != nil {
		t.Fatalf("category %s: %v", name, err)
	}
	return id
}

func (e *env) drafts(t *testing.T) map[string]asset.Asset {
	var list []asset.Asset
	e.do(t, http.MethodGet, "/api/assets?status=draft", nil, &list)
	byName := map[string]asset.Asset{}
	for _, a := range list {
		var full asset.Asset
		e.do(t, http.MethodGet, "/api/assets/"+strconv.FormatInt(a.ID, 10), nil, &full)
		byName[a.Name] = full
	}
	return byName
}

func TestScanCreatesDraftsAndAttachesNewFiles(t *testing.T) {
	e := setup(t)

	if code := e.do(t, http.MethodPost, "/api/scanner/scan", nil, nil); code != http.StatusBadRequest {
		t.Fatalf("scanning without roots should fail with 400, got %d", code)
	}

	// The Kipfel avatar is already in the library.
	avatarCat := e.categoryID(t, "Avatar")
	var kipfel asset.Asset
	e.do(t, http.MethodPost, "/api/assets", map[string]any{
		"name": "Kipfel_1.2.0", "category_id": avatarCat,
		"local_path": filepath.Join(e.lib, "Models", "Kipfel_1.2.0"),
		"booth_url":  "https://mukumi.booth.pm/items/5813187",
	}, &kipfel)

	cfg := scanner.DefaultConfig()
	cfg.Roots = []string{e.lib}
	if code := e.do(t, http.MethodPut, "/api/scanner/config", cfg, nil); code != http.StatusOK {
		t.Fatalf("save config failed: %d", code)
	}

	var result scanner.Result
	if code := e.do(t, http.MethodPost, "/api/scanner/scan", nil, &result); code != http.StatusOK {
		t.Fatalf("scan failed: %d", code)
	}
	if result.Created != 5 || len(result.Attached) != 1 || len(result.Warnings) != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !strings.HasSuffix(result.Attached[0].Path, "Kipfel v1.1.1.zip") || result.Attached[0].AssetID != kipfel.ID {
		t.Errorf("the archived Kipfel zip should attach to the existing avatar, got %+v", result.Attached)
	}

	drafts := e.drafts(t)
	for name := range drafts {
		if strings.Contains(strings.ToLower(name), "secret") || strings.Contains(name, "AvatarPass") {
			t.Fatalf("AvatarPass must never be scanned, got draft %q", name)
		}
	}

	lady, ok := drafts["hamanosis Small Lady Kipfel"]
	if !ok {
		t.Fatalf("missing Small Lady draft, got %v", keys(drafts))
	}
	if lady.Category == nil || lady.Category.Name != "Clothes" {
		t.Errorf("Small Lady category: %+v", lady.Category)
	}
	if lady.BoothURL != "https://booth.pm/ja/items/6834468" {
		t.Errorf("Small Lady should use the BOOTH .url link, got %q", lady.BoothURL)
	}
	if len(lady.Files) != 2 {
		t.Errorf("Small Lady should have folder + LEGACY zip, got %+v", lady.Files)
	}
	if len(lady.CompatibleAvatars) != 1 || lady.CompatibleAvatars[0].AvatarAssetID == nil || *lady.CompatibleAvatars[0].AvatarAssetID != kipfel.ID {
		t.Errorf("Small Lady should be compatible with Kipfel, got %+v", lady.CompatibleAvatars)
	}
	if !strings.Contains(string(lady.ScanInfo), "booth_source") {
		t.Errorf("scan_info should explain the BOOTH source, got %s", lady.ScanInfo)
	}

	hair := drafts["BukiyouTwinTail"]
	if hair.Category == nil || hair.Category.Name != "Hair" || len(hair.Files) != 2 {
		t.Errorf("BukiyouTwinTail should be Hair with the zip from Clothes/LEGACY: %+v / %+v", hair.Category, hair.Files)
	}

	piano := drafts["avatargimmick 8ya Undersea Piano 2"]
	if piano.BoothURL != "https://booth.pm/ja/items/4460917" || piano.Category == nil || piano.Category.Name != "Prop" {
		t.Errorf("piano draft: booth=%q category=%+v", piano.BoothURL, piano.Category)
	}

	meccha := drafts["Meccha Avatar"]
	if meccha.PreviewPath == "" {
		t.Errorf("Meccha should get main.png as preview")
	} else if _, err := os.Stat(filepath.Join(e.previews, filepath.Base(meccha.PreviewPath))); err != nil {
		t.Errorf("preview copy missing: %v", err)
	}

	if _, ok := drafts["song"]; !ok {
		t.Errorf("audio files should become drafts, got %v", keys(drafts))
	}

	// Re-scanning creates nothing new.
	var again scanner.Result
	e.do(t, http.MethodPost, "/api/scanner/scan", nil, &again)
	if again.Created != 0 || len(again.Attached) != 0 {
		t.Errorf("re-scan should be a no-op, got %+v", again)
	}

	// Ignore removes the draft and keeps it away on later scans.
	var ignoreRes map[string]int
	e.do(t, http.MethodPost, "/api/scanner/ignore", map[string]any{"asset_ids": []int64{hair.ID}}, &ignoreRes)
	if ignoreRes["ignored"] != 1 {
		t.Fatalf("ignore failed: %+v", ignoreRes)
	}
	e.do(t, http.MethodPost, "/api/scanner/scan", nil, &again)
	if again.Created != 0 || again.Ignored != 2 {
		t.Errorf("ignored paths must not come back, got %+v", again)
	}
	if _, err := os.Stat(filepath.Join(e.lib, "Hair", "BukiyouTwinTail_1.1.0")); err != nil {
		t.Errorf("ignoring must never touch files on disk: %v", err)
	}

	// Accept turns a draft into a library asset; accepting it again is refused.
	var acceptRes map[string]int
	e.do(t, http.MethodPost, "/api/scanner/accept", map[string]any{"asset_ids": []int64{lady.ID}}, &acceptRes)
	if acceptRes["accepted"] != 1 {
		t.Fatalf("accept failed: %+v", acceptRes)
	}
	if code := e.do(t, http.MethodPost, "/api/scanner/accept", map[string]any{"asset_ids": []int64{lady.ID}}, nil); code != http.StatusBadRequest {
		t.Errorf("accepting an active asset should be refused, got %d", code)
	}
	var stats asset.LibraryStats
	e.do(t, http.MethodGet, "/api/stats", nil, &stats)
	if stats.Total != 2 || stats.Drafts != 3 {
		t.Errorf("expected 2 active / 3 drafts, got %+v", stats)
	}

	// Un-ignoring brings the hair back on the next scan.
	var ignored []string
	e.do(t, http.MethodGet, "/api/scanner/ignored", nil, &ignored)
	for _, p := range ignored {
		e.do(t, http.MethodDelete, "/api/scanner/ignored?path="+urlQuery(p), nil, nil)
	}
	e.do(t, http.MethodPost, "/api/scanner/scan", nil, &again)
	if again.Created != 1 {
		t.Errorf("un-ignored asset should be scanned again, got %+v", again)
	}
}

func keys(m map[string]asset.Asset) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	return out
}

func urlQuery(s string) string {
	r := strings.NewReplacer("%", "%25", " ", "%20", "&", "%26", "+", "%2B", "#", "%23")
	return r.Replace(s)
}

func TestRegisteredFoldersGuideTheScan(t *testing.T) {
	e := setup(t)
	j := func(parts ...string) string { return filepath.Join(append([]string{e.lib}, parts...)...) }
	// A shop folder with two items, and a wrapper folder around one item.
	write(t, j("Clothes", "ShopB", "DressX", "Prefab", "DressX.prefab"), []byte("x"))
	write(t, j("Clothes", "ShopB", "DressY", "Prefab", "DressY.prefab"), []byte("x"))
	write(t, j("Accessory", "Halo", "Halo_Main", "Halo.prefab"), []byte("x"))
	write(t, j("Accessory", "Halo", "Halo_Main", "Textures", "halo.png"), pngBytes)

	// The user registers the whole shop folder as one asset, and the inner
	// Halo folder (not the wrapper) by hand.
	for _, path := range []string{j("Clothes", "ShopB"), j("Accessory", "Halo", "Halo_Main")} {
		if code := e.do(t, http.MethodPost, "/api/assets", map[string]any{"name": filepath.Base(path), "local_path": path}, nil); code != http.StatusCreated {
			t.Fatalf("create %s: %d", path, code)
		}
	}

	cfg := scanner.DefaultConfig()
	cfg.Roots = []string{e.lib}
	e.do(t, http.MethodPut, "/api/scanner/config", cfg, nil)
	var result scanner.Result
	e.do(t, http.MethodPost, "/api/scanner/scan", nil, &result)

	for name, d := range e.drafts(t) {
		if strings.Contains(d.LocalPath, "ShopB") || strings.Contains(d.LocalPath, "Halo") {
			t.Errorf("registered folders must not come back as drafts, got %q at %s", name, d.LocalPath)
		}
	}
	if result.AlreadyLinked != 2 {
		t.Errorf("both registered folders should count as already linked, got %+v", result)
	}
}

func TestNewFilesFindTheirExistingAsset(t *testing.T) {
	e := setup(t)
	j := func(parts ...string) string { return filepath.Join(append([]string{e.lib}, parts...)...) }
	clothes, accessory := e.categoryID(t, "Clothes"), e.categoryID(t, "Accessory")
	create := func(body map[string]any) asset.Asset {
		var a asset.Asset
		if code := e.do(t, http.MethodPost, "/api/assets", body, &a); code != http.StatusCreated {
			t.Fatalf("create %v: %d", body, code)
		}
		return a
	}

	// Renamed from BOOTH; its old version folder has since been deleted.
	dress := create(map[string]any{"name": "リボンドレス", "category_id": clothes, "local_path": j("Clothes", "Ribbon_Dress_1.0")})
	write(t, j("Clothes", "Ribbon_Dress_1.1", "Dress.unitypackage"), []byte("x"))
	// Registered by BOOTH link only, now downloaded under a different name.
	wings := create(map[string]any{"name": "Angel Wings", "category_id": accessory, "booth_url": "https://booth.pm/ja/items/1234567"})
	write(t, j("Accessory", "1234567 tenshi_hane", "Wings.unitypackage"), []byte("x"))
	// Only the display name matches: suggested, not linked.
	halo := create(map[string]any{"name": "Halo", "category_id": accessory})
	write(t, j("Accessory", "Halo", "Halo.unitypackage"), []byte("x"))
	write(t, j("Accessory", "Halo", "main.png"), pngBytes)

	cfg := scanner.DefaultConfig()
	cfg.Roots = []string{e.lib}
	e.do(t, http.MethodPut, "/api/scanner/config", cfg, nil)
	var result scanner.Result
	e.do(t, http.MethodPost, "/api/scanner/scan", nil, &result)

	attached := map[int64]string{}
	for _, a := range result.Attached {
		attached[a.AssetID] = a.Path
	}
	if !strings.HasSuffix(attached[dress.ID], "Ribbon_Dress_1.1") {
		t.Errorf("new dress version should attach to the renamed asset, got %+v", result.Attached)
	}
	if !strings.HasSuffix(attached[wings.ID], "1234567 tenshi_hane") {
		t.Errorf("BOOTH id in the folder name should attach to the wings, got %+v", result.Attached)
	}

	drafts := e.drafts(t)
	haloDraft, ok := drafts["Halo"]
	if !ok {
		t.Fatalf("Halo should stay a draft, got %v", keys(drafts))
	}
	if !strings.Contains(string(haloDraft.ScanInfo), `"merge_candidates":[{"id":`+strconv.FormatInt(halo.ID, 10)) {
		t.Errorf("Halo draft should suggest the existing Halo, got %s", haloDraft.ScanInfo)
	}

	// Merging moves the files and preview into the existing asset.
	var merged asset.Asset
	if code := e.do(t, http.MethodPost, "/api/scanner/merge", map[string]any{"draft_id": haloDraft.ID, "target_id": halo.ID}, &merged); code != http.StatusOK {
		t.Fatalf("merge failed: %d", code)
	}
	if len(merged.Files) != 1 || merged.LocalPath != j("Accessory", "Halo") || merged.PreviewPath == "" {
		t.Errorf("merged asset: files=%+v local=%q preview=%q", merged.Files, merged.LocalPath, merged.PreviewPath)
	}
	if code := e.do(t, http.MethodGet, "/api/assets/"+strconv.FormatInt(haloDraft.ID, 10), nil, nil); code != http.StatusNotFound {
		t.Errorf("merged draft should be gone, got %d", code)
	}
	if code := e.do(t, http.MethodPost, "/api/scanner/merge", map[string]any{"draft_id": dress.ID, "target_id": halo.ID}, nil); code != http.StatusBadRequest {
		t.Errorf("only drafts can be merged, got %d", code)
	}
}
