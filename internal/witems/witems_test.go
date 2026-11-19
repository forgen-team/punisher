package witems

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/forgen-team/punisher/internal/services"
)

// fakeAuth answers /v1/auth/me: token "owner" → owner-id, "other" → other-id, "down" → 500, else 401.
func fakeAuth(t *testing.T, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		c, _ := r.Cookie(sessionCookie)
		switch {
		case c == nil:
			w.WriteHeader(http.StatusUnauthorized)
		case c.Value == "owner":
			_, _ = w.Write([]byte(`{"user":{"id":"owner-id"}}`))
		case c.Value == "other":
			_, _ = w.Write([]byte(`{"user":{"id":"other-id"}}`))
		case c.Value == "down":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func guarded(g *Guard) http.Handler {
	return g.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
}

func call(h http.Handler, token, origin string) (int, string) {
	req := httptest.NewRequest(http.MethodPost, "/v1/lectures/instant", nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body.Error.Code
}

func TestGuard(t *testing.T) {
	var calls atomic.Int32
	auth := fakeAuth(t, &calls)
	g := &Guard{AuthServiceURL: auth.URL, OwnerUserID: "owner-id", Origin: "https://witems.kr"}
	h := guarded(g)
	cases := []struct {
		name, token, origin, code string
		status                    int
	}{
		{"owner", "owner", "https://witems.kr", "", http.StatusNoContent},
		{"owner without Origin (same-origin GET-like)", "owner", "", "", http.StatusNoContent},
		{"no cookie", "", "", "UNAUTHORIZED", http.StatusUnauthorized},
		{"rejected session", "expired", "", "UNAUTHORIZED", http.StatusUnauthorized},
		{"another account", "other", "", "OWNER_ONLY", http.StatusForbidden},
		{"foreign origin", "owner", "https://evil.example", "FORBIDDEN_ORIGIN", http.StatusForbidden},
		{"auth service failing is not allowed", "down", "", "AUTH_UNAVAILABLE", http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, code := call(h, tc.token, tc.origin)
			if status != tc.status || code != tc.code {
				t.Fatalf("got %d %q, want %d %q", status, code, tc.status, tc.code)
			}
		})
	}
}

func TestGuardCachesOnlySuccess(t *testing.T) {
	var calls atomic.Int32
	auth := fakeAuth(t, &calls)
	now := time.Unix(0, 0)
	g := &Guard{AuthServiceURL: auth.URL, OwnerUserID: "owner-id", Origin: "https://witems.kr", Now: func() time.Time { return now }}
	h := guarded(g)
	call(h, "owner", "")
	call(h, "owner", "")
	if calls.Load() != 1 {
		t.Fatalf("owner session checked %d times, want 1 (cached)", calls.Load())
	}
	now = now.Add(sessionTTL + time.Second)
	call(h, "owner", "")
	if calls.Load() != 2 {
		t.Fatalf("expired cache entry not rechecked")
	}
	call(h, "expired", "")
	call(h, "expired", "")
	if calls.Load() != 4 {
		t.Fatalf("rejected sessions must not be cached, calls=%d", calls.Load())
	}
}

func TestAppHandler(t *testing.T) {
	files := fstest.MapFS{
		"remoteEntry.js":           {Data: []byte("export const x = 1")},
		"assets/mount-D4OSuUjC.js": {Data: []byte("m")},
		"assets/notes.txt":         {Data: []byte("t")},
		"assets/mf-manifest.json":  {Data: []byte("{}")},
		"index.html":               {Data: []byte("<html>")},
	}
	h := appHandler(files, "1.2.3")
	get := func(path string, header ...string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		for i := 0; i+1 < len(header); i += 2 {
			req.Header.Set(header[i], header[i+1])
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	if rec := get("/healthz"); rec.Code != http.StatusOK {
		t.Fatalf("healthz %d", rec.Code)
	}
	rec := get("/manifest.json")
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil || m["name"] != "punisher" || m["version"] != "1.2.3" || m["api"] != "/apps-api/punisher/" {
		t.Fatalf("manifest %v %v", m, err)
	}

	entry := get("/static/remoteEntry.js")
	if entry.Code != http.StatusOK || entry.Header().Get("Cache-Control") != Revalidate || entry.Header().Get("Content-Type") != "text/javascript; charset=utf-8" {
		t.Fatalf("remoteEntry %d %v", entry.Code, entry.Header())
	}
	if again := get("/static/remoteEntry.js", "If-None-Match", entry.Header().Get("ETag")); again.Code != http.StatusNotModified {
		t.Fatalf("conditional request got %d, want 304", again.Code)
	}
	if hashed := get("/static/assets/mount-D4OSuUjC.js"); hashed.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Fatalf("hashed asset cache %q", hashed.Header().Get("Cache-Control"))
	}
	if named := get("/static/assets/mf-manifest.json"); named.Header().Get("Cache-Control") != Revalidate {
		t.Fatalf("a hyphenated name is not a hash: %q", named.Header().Get("Cache-Control"))
	}
	for _, p := range []string{"/static/index.html", "/static/assets/notes.txt", "/static/missing.js"} {
		if rec := get(p); rec.Code != http.StatusNotFound {
			t.Fatalf("%s: %d, want 404", p, rec.Code)
		}
	}
	// A path climbing out of /static is never served (the mux redirects to the cleaned path).
	if rec := get("/static/../app.go"); rec.Code == http.StatusOK {
		t.Fatalf("/static/../app.go served: %q", rec.Body.String())
	}
}

type recordSignals struct{ got string }

func (r *recordSignals) LearningSignals(_ context.Context, userID string) (services.LearningSignals, error) {
	r.got = userID
	return services.LearningSignals{}, nil
}

func TestOwnerSignalsAsksForTheOwner(t *testing.T) {
	rec := &recordSignals{}
	if _, err := (OwnerSignals{Source: rec, OwnerUserID: "owner-id"}).LearningSignals(context.Background(), "local"); err != nil {
		t.Fatal(err)
	}
	if rec.got != "owner-id" {
		t.Fatalf("rss-server asked for %q, want the Witems owner id", rec.got)
	}
}
