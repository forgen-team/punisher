// Package webui serves the built frontend (web/ → internal/webui/dist,
// produced by `make web`) embedded into the binary.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var dist embed.FS

const placeholder = `<!doctype html><html lang="ko"><meta charset="utf-8"><title>Punisher</title>
<body style="font-family:system-ui;padding:2rem">
<h1>프론트엔드가 포함되지 않은 빌드입니다</h1>
<p><code>make build</code> 로 다시 빌드해 주세요 (web/ 빌드 결과가 바이너리에 들어갑니다).</p></body></html>`

// Handler serves static assets and falls back to index.html for client-side
// routes (React Router). Hashed assets are cached; index.html is not.
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" && name != "index.html" {
			if fi, err := fs.Stat(sub, name); err == nil && !fi.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, r)
				return
			}
			if strings.HasPrefix(name, "assets/") || path.Ext(name) != "" {
				http.NotFound(w, r)
				return
			}
		}
		index, err := fs.ReadFile(sub, "index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		if err != nil {
			_, _ = w.Write([]byte(placeholder))
			return
		}
		_, _ = w.Write(index)
	})
}
