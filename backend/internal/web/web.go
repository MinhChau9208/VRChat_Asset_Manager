// Package web serves the exported frontend (frontend/out) from the release
// binary, so a user only has to start one program.
//
// The UI is embedded only when building with `-tags release` (see
// embed_release.go and scripts/build-release.ps1). Development builds embed
// nothing and the UI runs under `next dev` as before.
package web

import (
	"io/fs"
	"net/http"
	"os/exec"
	"path"
	"runtime"
	"strings"
)

// dist holds the exported UI; nil unless built with the release tag.
var dist fs.FS

// FS returns the embedded UI, or nil in development builds.
func FS() fs.FS { return dist }

// Enabled reports whether this binary carries the UI (a release build).
func Enabled() bool { return dist != nil }

// NewHandler serves a Next.js static export. A route such as /assets maps to
// assets.html; unknown paths get 404.html.
func NewHandler(fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		// Unknown API routes must not answer with an HTML page.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		switch {
		case name == "":
			name = "index.html"
		case isFile(fsys, name):
		case isFile(fsys, name+".html"):
			name += ".html"
		default:
			serveNotFound(w, fsys)
			return
		}

		if strings.HasPrefix(name, "_next/static/") {
			// Content-hashed file names never change.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			// Pages must pick up a new version right after an update.
			w.Header().Set("Cache-Control", "no-cache")
		}
		http.ServeFileFS(w, r, fsys, name)
	})
}

func isFile(fsys fs.FS, name string) bool {
	info, err := fs.Stat(fsys, name)
	return err == nil && !info.IsDir()
}

func serveNotFound(w http.ResponseWriter, fsys fs.FS) {
	page, err := fs.ReadFile(fsys, "404.html")
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(page)
}

// OpenBrowser opens url in the user's default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
