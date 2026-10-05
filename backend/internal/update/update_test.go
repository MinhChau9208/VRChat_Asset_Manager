package update_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"vrchat-asset-manager/backend/internal/update"
)

func fakeGitHub(t *testing.T, tag string, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `","html_url":"https://example.test/releases/` + tag + `"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestInfo(t *testing.T) {
	tests := []struct {
		current, latest string
		available       bool
		wantCalls       int32
	}{
		{"1.0.0", "v1.1.0", true, 1},
		{"v1.2", "v1.2.0", false, 1},
		{"1.10.0", "v1.9.9", false, 1},
		{"2.0.0", "garbage", false, 1},
		{"dev", "v9.0.0", false, 0},           // development build: no network
		{"516495c-dirty", "v9.0.0", false, 0}, // untagged build: no network
	}
	for _, tt := range tests {
		var calls atomic.Int32
		c := update.NewChecker(tt.current, "owner/repo")
		c.SetAPIURL(fakeGitHub(t, tt.latest, &calls).URL)

		info := c.Info(context.Background())
		if info.UpdateAvailable != tt.available {
			t.Errorf("%s vs %s: update_available %v, want %v", tt.current, tt.latest, info.UpdateAvailable, tt.available)
		}
		if tt.available && (info.Latest != "1.1.0" || info.URL == "") {
			t.Errorf("%s vs %s: got %+v", tt.current, tt.latest, info)
		}
		if info.Version != tt.current {
			t.Errorf("version %q, want %q", info.Version, tt.current)
		}
		if calls.Load() != tt.wantCalls {
			t.Errorf("%s: %d GitHub calls, want %d", tt.current, calls.Load(), tt.wantCalls)
		}
	}
}

func TestInfoIsCached(t *testing.T) {
	var calls atomic.Int32
	c := update.NewChecker("1.0.0", "owner/repo")
	c.SetAPIURL(fakeGitHub(t, "v1.1.0", &calls).URL)
	for range 3 {
		c.Info(context.Background())
	}
	if calls.Load() != 1 {
		t.Fatalf("%d GitHub calls, want 1", calls.Load())
	}
}

func TestOfflineIsQuiet(t *testing.T) {
	c := update.NewChecker("1.0.0", "owner/repo")
	c.SetAPIURL("http://127.0.0.1:1/unreachable")
	if info := c.Info(context.Background()); info.UpdateAvailable || info.Version != "1.0.0" {
		t.Fatalf("got %+v", info)
	}
}
