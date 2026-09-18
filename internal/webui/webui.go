// Package webui embeds mcm's built SvelteKit dashboard (source in
// ../../web) so `mcm web` can serve it straight from the binary — no
// separate process or asset directory to ship, per the packaging
// decision in RESEARCH.md §7.6. Run `npm run build` in web/ to
// regenerate dist/ before building mcm.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Handler serves the built dashboard. It's a single-page app (SvelteKit
// in SPA mode, client-side routing under /servers/...), so any request
// that isn't a real static asset falls back to index.html rather than
// 404ing.
func Handler() (http.Handler, error) {
	assets, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}

	index, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.FS(assets))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(assets, strings.TrimPrefix(r.URL.Path, "/")); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(index)
			return
		}
		fileServer.ServeHTTP(w, r)
	}), nil
}
