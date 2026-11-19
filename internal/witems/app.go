package witems

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"path"
	"regexp"
	"strings"
)

// dist is the Module Federation remote build (web: `npm run build:witems`,
// copied here by `make witems-web`). It is empty in normal builds.
//
//go:embed all:dist
var dist embed.FS

// Manifest is the app's entry in the Witems shell (contract §2).
var Manifest = map[string]any{
	"contract":    1,
	"name":        "punisher",
	"title":       "학습",
	"nav":         map[string]any{"label": "학습", "icon": "graduation-cap", "order": 10},
	"remoteEntry": "/apps-static/punisher/remoteEntry.js",
	"api":         "/apps-api/punisher/",
	"health":      "/healthz",
}

// Revalidate is for files whose name does not change between builds
// (remoteEntry.js). `no-cache` alone is rewritten by Cloudflare's browser
// cache TTL, so private and max-age=0 go with it (contract §6).
const Revalidate = "private, no-cache, max-age=0, must-revalidate"

// hashedName matches Vite's "-XXXXXXXX.ext". A real hash mixes cases/digits;
// requiring one non-lowercase char keeps words like "-manifest" out (a
// lowercase-only hash is just revalidated, which is safe).
var hashedName = regexp.MustCompile(`-([A-Za-z0-9_-]{8})\.[a-z0-9]+$`)

func isHashedName(base string) bool {
	m := hashedName.FindStringSubmatch(base)
	return m != nil && strings.ContainsFunc(m[1], func(r rune) bool { return r < 'a' || r > 'z' })
}

// served file types; anything else under /static is 404.
var staticTypes = map[string]string{
	".js": "text/javascript; charset=utf-8", ".mjs": "text/javascript; charset=utf-8",
	".css": "text/css; charset=utf-8", ".json": "application/json", ".map": "application/json",
	".svg": "image/svg+xml", ".png": "image/png", ".webp": "image/webp", ".woff2": "font/woff2",
}

// AppHandler serves /static/*, /manifest.json and /healthz — everything the
// shell needs that is not the /v1 API. None of it needs a login.
func AppHandler(version string) http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return appHandler(sub, version)
}

func appHandler(sub fs.FS, version string) http.Handler {
	etags := computeETags(sub)
	if _, ok := etags["remoteEntry.js"]; !ok {
		log.Printf("witems: no remoteEntry.js in the embedded build — run `make witems-web` (the shell will show the load-failed screen)")
	}
	manifest := map[string]any{"version": version}
	for k, v := range Manifest {
		manifest[k] = v
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", Revalidate)
		_, _ = w.Write(manifestJSON)
	})
	files := http.FileServer(http.FS(sub))
	mux.Handle("/static/", http.StripPrefix("/static", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		ctype, ok := staticTypes[path.Ext(name)]
		etag, exists := etags[name]
		if !ok || !exists {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ctype)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("ETag", etag)
		if strings.HasPrefix(name, "assets/") && isHashedName(path.Base(name)) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", Revalidate)
		}
		files.ServeHTTP(w, r) // honours If-None-Match against the ETag set above
	})))
	return mux
}

// computeETags hashes every embedded file once (the build is immutable).
func computeETags(sub fs.FS) map[string]string {
	out := map[string]string{}
	_ = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(path.Base(p), ".") {
			return nil
		}
		data, err := fs.ReadFile(sub, p)
		if err != nil {
			return nil
		}
		sum := sha256.Sum256(data)
		out[p] = `"` + hex.EncodeToString(sum[:8]) + `"`
		return nil
	})
	return out
}

func init() {
	// Some minimal containers have no /etc/mime.types; keep .js right for module scripts.
	_ = mime.AddExtensionType(".js", "text/javascript; charset=utf-8")
}
