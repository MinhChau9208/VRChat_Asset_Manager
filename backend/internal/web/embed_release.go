//go:build release

package web

import (
	"embed"
	"io/fs"
)

// dist/ is a copy of frontend/out made by scripts/build-release.ps1.
//
//go:embed all:dist
var embedded embed.FS

func init() {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}
	dist = sub
}
