package booth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"vrchat-asset-manager/backend/internal/asset"
	"vrchat-asset-manager/backend/internal/database"
	"vrchat-asset-manager/backend/migrations"
)

var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

// Synthetic items shaped like BOOTH's JSON.
func fakeItems(imageBase string) map[string]map[string]any {
	img := func(name string) map[string]any {
		return map[string]any{"original": imageBase + "/" + name, "resized": imageBase + "/small/" + name}
	}
	return map[string]map[string]any{
		"1000001": {
			"id": 1000001, "name": "Test Avatar / オリジナル3Dモデル", "url": "https://shop.booth.pm/items/1000001",
			"category": map[string]any{"name": "3Dキャラクター", "parent": map[string]any{"name": "3Dモデル"}},
			"shop":     map[string]any{"name": "Avatar Shop"},
			"images":   []any{img("avatar.png")},
		},
		"2000002": {
			"id": 2000002, "name": "Ribbon Dress 【衣装】", "url": "https://dress.booth.pm/items/2000002",
			"description": "かわいいドレス\n\n■ 対応アバター\n・Test Avatar − https://shop.booth.pm/items/1000001\n・マヌカ Manuka\n※ 他のアバターは自己責任\n\n■ 内容\n・FBX",
			"category":    map[string]any{"name": "3D衣装", "parent": map[string]any{"name": "3Dモデル"}},
			"shop":        map[string]any{"name": "Dress Shop"},
			"tags":        []any{map[string]any{"name": "3D"}, map[string]any{"name": "VRChat"}, map[string]any{"name": "ドレス"}, map[string]any{"name": "リボン"}},
			"variations":  []any{map[string]any{"name": "Test Avatar用"}},
			"images":      []any{img("dress.png"), img("dress2.png")},
		},
		"3000003": {
			"id": 3000003, "name": "ふわふわツインテール ヘア", "url": "https://hair.booth.pm/items/3000003",
			"category": map[string]any{"name": "3D装飾品", "parent": map[string]any{"name": "3Dモデル"}},
			"shop":     map[string]any{"name": "Hair Shop"},
		},
		"4000004": {
			"id": 4000004, "name": "Sparkle Eye Texture 瞳テクスチャ", "url": "https://eye.booth.pm/items/4000004",
			"category": map[string]any{"name": "3Dテクスチャ", "parent": map[string]any{"name": "3Dモデル"}},
			"shop":     map[string]any{"name": "Eye Shop"},
		},
	}
}

type testEnv struct {
	mux      *http.ServeMux
	db       *database.DB
	hits     *atomic.Int32
	previews string
}

func setup(t *testing.T) *testEnv {
	t.Helper()
	var hits atomic.Int32
	var items map[string]map[string]any
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/img/") {
			_, _ = w.Write(pngBytes)
			return
		}
		hits.Add(1)
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/ja/items/"), ".json")
		item, ok := items[id]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(item)
	}))
	t.Cleanup(fake.Close)
	items = fakeItems(fake.URL + "/img")

	tmp := t.TempDir()
	db, err := database.Connect(filepath.Join(tmp, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(migrations.FS); err != nil {
		t.Fatal(err)
	}

	client := NewClient(db.DB)
	client.baseURL = fake.URL
	client.allowImageHost = func(host string) bool { return host == "127.0.0.1" }
	client.last = client.last.Add(-minInterval) // no initial wait

	previews := filepath.Join(tmp, "previews")
	repo := asset.NewRepository(db.DB)
	mux := http.NewServeMux()
	asset.NewHandler(repo).RegisterRoutes(mux)
	NewHandler(NewService(db.DB, client, repo, previews)).RegisterRoutes(mux)
	return &testEnv{mux: mux, db: db, hits: &hits, previews: previews}
}

func (e *testEnv) do(t *testing.T, method, target string, body, out any) int {
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

func (e *testEnv) categoryID(t *testing.T, name string) int64 {
	var id int64
	if err := e.db.QueryRow("SELECT id FROM categories WHERE name = ?", name).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestParseItemID(t *testing.T) {
	ok := map[string]string{
		"https://booth.pm/ja/items/6834468":                 "6834468",
		"https://booth.pm/en/items/4394473":                 "4394473",
		"https://hamanosis.booth.pm/items/6834468?_gl=1*ab": "6834468",
		" 5813187 ": "5813187",
	}
	for in, want := range ok {
		if got, err := ParseItemID(in); err != nil || got != want {
			t.Errorf("ParseItemID(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "https://booth.pm/", "https://example.com/items/123", "hello"} {
		if _, err := ParseItemID(bad); err == nil {
			t.Errorf("ParseItemID(%q) should fail", bad)
		}
	}
}

func TestLookupSuggestionsAndCache(t *testing.T) {
	e := setup(t)
	var avatar asset.Asset
	e.do(t, http.MethodPost, "/api/assets", map[string]any{
		"name": "Test Avatar", "category_id": e.categoryID(t, "Avatar"),
		"booth_url": "https://shop.booth.pm/items/1000001",
	}, &avatar)

	var s Suggestion
	if code := e.do(t, http.MethodGet, "/api/booth/lookup?url=https://dress.booth.pm/items/2000002", nil, &s); code != http.StatusOK {
		t.Fatalf("lookup failed: %d", code)
	}
	if s.Name != "Ribbon Dress 【衣装】" || s.Author != "Dress Shop" || s.BoothURL != "https://booth.pm/ja/items/2000002" {
		t.Errorf("unexpected basics: %+v", s)
	}
	if s.CategoryName != "Clothes" || s.BoothCategory != "3Dモデル > 3D衣装" {
		t.Errorf("unexpected category: %q / %q", s.CategoryName, s.BoothCategory)
	}
	if strings.Join(s.Tags, ",") != "ドレス,リボン" {
		t.Errorf("generic tags should be dropped, got %v", s.Tags)
	}
	if len(s.Images) != 2 {
		t.Errorf("expected 2 images, got %v", s.Images)
	}
	if len(s.CompatibleAvatars) != 2 ||
		s.CompatibleAvatars[0].AvatarAssetID == nil || *s.CompatibleAvatars[0].AvatarAssetID != avatar.ID ||
		s.CompatibleAvatars[1].AvatarName != "マヌカ Manuka" || s.CompatibleAvatars[1].AvatarAssetID != nil {
		t.Errorf("expected library avatar + Manuka by name, got %+v", s.CompatibleAvatars)
	}

	// Served from cache the second time; refresh=1 goes to BOOTH again.
	before := e.hits.Load()
	e.do(t, http.MethodGet, "/api/booth/lookup?url=2000002", nil, nil)
	if e.hits.Load() != before {
		t.Errorf("second lookup should come from the cache")
	}
	e.do(t, http.MethodGet, "/api/booth/lookup?url=2000002&refresh=1", nil, nil)
	if e.hits.Load() != before+1 {
		t.Errorf("refresh should fetch again")
	}

	if code := e.do(t, http.MethodGet, "/api/booth/lookup?url=9999999", nil, nil); code != http.StatusNotFound {
		t.Errorf("missing item should be 404, got %d", code)
	}
	if code := e.do(t, http.MethodGet, "/api/booth/lookup?url=https://example.com/x", nil, nil); code != http.StatusBadRequest {
		t.Errorf("invalid URL should be 400, got %d", code)
	}

	// An avatar item is not "compatible with itself".
	e.do(t, http.MethodGet, "/api/booth/lookup?url=1000001", nil, &s)
	if s.CategoryName != "Avatar" || len(s.CompatibleAvatars) != 0 {
		t.Errorf("avatar item: category %q compat %+v", s.CategoryName, s.CompatibleAvatars)
	}
}

func TestCategoryRefinements(t *testing.T) {
	e := setup(t)
	for id, want := range map[string]string{"3000003": "Hair", "4000004": "Eyes"} {
		var s Suggestion
		e.do(t, http.MethodGet, "/api/booth/lookup?url="+id, nil, &s)
		if s.CategoryName != want {
			t.Errorf("item %s: expected %s, got %q", id, want, s.CategoryName)
		}
	}
}

func TestApplyFillsDrafts(t *testing.T) {
	e := setup(t)
	e.do(t, http.MethodPost, "/api/assets", map[string]any{
		"name": "Test Avatar", "category_id": e.categoryID(t, "Avatar"),
		"booth_url": "https://shop.booth.pm/items/1000001",
	}, nil)

	var draft, noLink asset.Asset
	e.do(t, http.MethodPost, "/api/assets", map[string]any{
		"name": "ribbon_dress_v2", "status": "draft", "booth_url": "https://booth.pm/ja/items/2000002",
	}, &draft)
	e.do(t, http.MethodPost, "/api/assets", map[string]any{"name": "mystery", "status": "draft"}, &noLink)

	var resp struct {
		Results []ApplyResult `json:"results"`
	}
	e.do(t, http.MethodPost, "/api/booth/apply", map[string]any{"asset_ids": []int64{draft.ID, noLink.ID}, "include_tags": true}, &resp)
	if len(resp.Results) != 2 || !resp.Results[0].OK || resp.Results[1].OK || resp.Results[1].Error != "no BOOTH link" {
		t.Fatalf("unexpected results: %+v", resp.Results)
	}

	var got asset.Asset
	e.do(t, http.MethodGet, fmt.Sprintf("/api/assets/%d", draft.ID), nil, &got)
	if got.Name != "Ribbon Dress 【衣装】" || got.Author != "Dress Shop" || got.Category == nil || got.Category.Name != "Clothes" {
		t.Errorf("fields not applied: %+v", got)
	}
	if len(got.CompatibleAvatars) != 2 || len(got.Tags) != 2 {
		t.Errorf("compat/tags not merged: %+v / %v", got.CompatibleAvatars, got.Tags)
	}
	if got.Status != "draft" {
		t.Errorf("apply must not accept the draft, got status %q", got.Status)
	}
	if got.PreviewPath == "" {
		t.Fatalf("preview should be downloaded")
	}
	if _, err := os.Stat(filepath.Join(e.previews, filepath.Base(got.PreviewPath))); err != nil {
		t.Errorf("preview file missing: %v", err)
	}

	// Existing author/category are kept on a second apply.
	e.do(t, http.MethodPut, fmt.Sprintf("/api/assets/%d", draft.ID), map[string]any{
		"name": got.Name, "author": "Me", "category_id": e.categoryID(t, "Outfit"), "booth_url": got.BoothURL,
	}, nil)
	e.do(t, http.MethodPost, "/api/booth/apply", map[string]any{"asset_ids": []int64{draft.ID}}, &resp)
	e.do(t, http.MethodGet, fmt.Sprintf("/api/assets/%d", draft.ID), nil, &got)
	if got.Author != "Me" || got.Category.Name != "Outfit" {
		t.Errorf("apply should not overwrite author/category: %q / %q", got.Author, got.Category.Name)
	}
}

func TestPreviewFromURLOnlyAllowsBoothImages(t *testing.T) {
	e := setup(t)
	var a asset.Asset
	e.do(t, http.MethodPost, "/api/assets", map[string]any{"name": "X"}, &a)
	if code := e.do(t, http.MethodPost, "/api/booth/preview", map[string]any{"asset_id": a.ID, "url": "https://evil.example.com/a.png"}, nil); code != http.StatusBadRequest {
		t.Errorf("non-BOOTH image host should be refused, got %d", code)
	}
}

func TestAvatarAliasesAndMentions(t *testing.T) {
	aliases := AvatarAliases("Kipfel_1.2.0", "キプフェル Kipfel / オリジナル3Dモデル")
	if strings.Join(aliases, ",") != "Kipfel,キプフェル" {
		t.Fatalf("unexpected aliases %v", aliases)
	}
	kipfel := Avatar{ID: 1, Name: "Kipfel_1.2.0", Aliases: aliases}
	for text, want := range map[string]bool{
		"【キプフェル対応】ツインテール":       true, // Japanese: substring
		"For Kipfel and Manuka": true, // Latin: whole word
		"Kipfelsuit outfit":     false,
		"Manuka only":           false,
	} {
		if got := mentions(text, kipfel); got != want {
			t.Errorf("mentions(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestAvatarAliasesSkipCommonWords(t *testing.T) {
	if got := AvatarAliases("New test"); len(got) != 0 {
		t.Errorf("generic names must not become aliases, got %v", got)
	}
	if got := AvatarAliases("ししゅか siska / オリジナル3Dモデル"); strings.Join(got, ",") != "ししゅか,siska" {
		t.Errorf("unexpected aliases %v", got)
	}
}
