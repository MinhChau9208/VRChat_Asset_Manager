package web_test

import (
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"vrchat-asset-manager/backend/internal/web"
)

func TestHandler(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":               {Data: []byte("library")},
		"assets.html":              {Data: []byte("asset detail")},
		"assets/new.html":          {Data: []byte("new asset")},
		"404.html":                 {Data: []byte("not found page")},
		"_next/static/chunks/a.js": {Data: []byte("js")},
	}
	h := web.NewHandler(fsys)

	tests := []struct {
		method, path string
		status       int
		body         string
		cache        string
	}{
		{"GET", "/", 200, "library", "no-cache"},
		{"GET", "/assets?id=5", 200, "asset detail", "no-cache"},
		{"GET", "/assets/new", 200, "new asset", "no-cache"},
		{"GET", "/_next/static/chunks/a.js", 200, "js", "immutable"},
		{"GET", "/nope", 404, "not found page", ""},
		{"GET", "/../../etc/passwd", 404, "not found page", ""},
		{"GET", "/api/unknown", 404, "404 page not found", ""},
		{"POST", "/", 405, "Method Not Allowed", ""},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
		if rec.Code != tt.status {
			t.Errorf("%s %s: status %d, want %d", tt.method, tt.path, rec.Code, tt.status)
		}
		if !strings.Contains(rec.Body.String(), tt.body) {
			t.Errorf("%s %s: body %q, want it to contain %q", tt.method, tt.path, rec.Body.String(), tt.body)
		}
		if tt.cache != "" && !strings.Contains(rec.Header().Get("Cache-Control"), tt.cache) {
			t.Errorf("%s %s: Cache-Control %q, want %q", tt.method, tt.path, rec.Header().Get("Cache-Control"), tt.cache)
		}
	}
}

func TestDevBuildEmbedsNothing(t *testing.T) {
	if web.Enabled() {
		t.Skip("release build")
	}
	if web.FS() != nil {
		t.Fatal("FS() should be nil without the release tag")
	}
}
