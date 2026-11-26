package witems

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

func TestGuardServiceCalls(t *testing.T) {
	var calls atomic.Int32
	auth := fakeAuth(t, &calls)
	g := &Guard{AuthServiceURL: auth.URL, OwnerUserID: "owner-id", Origin: "https://witems.kr", InternalToken: "secret"}
	h := guarded(g)
	serve := func(token, user string) int {
		req := httptest.NewRequest(http.MethodPost, "/v1/study-requests", nil)
		req.Header.Set("X-Internal-Token", token)
		req.Header.Set("X-Internal-User-Id", user)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if got := serve("secret", "owner-id"); got != http.StatusNoContent {
		t.Fatalf("owner service call: %d", got)
	}
	if got := serve("wrong", "owner-id"); got != http.StatusUnauthorized {
		t.Fatalf("wrong token: %d", got)
	}
	if got := serve("secret", "other-id"); got != http.StatusForbidden {
		t.Fatalf("another user via service call: %d", got)
	}
	if calls.Load() != 0 {
		t.Fatal("service calls must not consult the session service")
	}
	// no token configured: service calls are off even with an empty header match
	off := guarded(&Guard{AuthServiceURL: auth.URL, OwnerUserID: "owner-id", Origin: "https://witems.kr"})
	req := httptest.NewRequest(http.MethodGet, "/v1/today", nil)
	req.Header.Set("X-Internal-User-Id", "owner-id")
	rec := httptest.NewRecorder()
	off.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("service call without a configured token: %d", rec.Code)
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
	if calls.Load() != 3 {
		t.Fatalf("a rejected session is remembered for a moment, calls=%d", calls.Load())
	}
	now = now.Add(rejectedTTL + time.Second)
	call(h, "expired", "")
	if calls.Load() != 4 {
		t.Fatalf("a rejected session is checked again after rejectedTTL, calls=%d", calls.Load())
	}
	call(h, "down", "")
	call(h, "down", "")
	if calls.Load() != 6 {
		t.Fatalf("auth service failures must not be cached, calls=%d", calls.Load())
	}
}

func TestGuardSharesConcurrentChecks(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		<-release
		_, _ = w.Write([]byte(`{"user":{"id":"owner-id"}}`))
	}))
	t.Cleanup(auth.Close)
	h := guarded(&Guard{AuthServiceURL: auth.URL, OwnerUserID: "owner-id", Origin: "https://witems.kr"})
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() { defer wg.Done(); call(h, "owner", "") }()
	}
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("5 concurrent requests made %d auth calls, want 1", calls.Load())
	}
}

func TestGuardRedirectIsNotSignedIn(t *testing.T) {
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	}))
	t.Cleanup(auth.Close)
	h := guarded(&Guard{AuthServiceURL: auth.URL, OwnerUserID: "owner-id", Origin: "https://witems.kr"})
	if status, code := call(h, "owner", ""); status != http.StatusUnauthorized || code != "UNAUTHORIZED" {
		t.Fatalf("got %d %q", status, code)
	}
}

func TestGuardOrigins(t *testing.T) {
	var calls atomic.Int32
	auth := fakeAuth(t, &calls)
	h := guarded(&Guard{AuthServiceURL: auth.URL, OwnerUserID: "owner-id", Origin: "https://witems.kr, https://www.witems.kr"})
	for origin, want := range map[string]int{
		"https://witems.kr":      http.StatusNoContent,
		"https://WWW.witems.kr":  http.StatusNoContent,
		"null":                   http.StatusForbidden,
		"https://witems.kr.evil": http.StatusForbidden,
	} {
		if status, _ := call(h, "owner", origin); status != want {
			t.Errorf("Origin %q: %d, want %d", origin, status, want)
		}
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

	// nav.items (contract §2): the shell's sidebar sub-menu — at most 8, each href/match an in-app hash "#/…".
	nav, _ := m["nav"].(map[string]any)
	items, _ := nav["items"].([]any)
	var hrefs []string
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		href, _ := item["href"].(string)
		if label, _ := item["label"].(string); label == "" || !strings.HasPrefix(href, "#/") {
			t.Fatalf("nav item %v: needs a label and a #/ href", item)
		}
		match, _ := item["match"].([]any)
		for _, p := range match {
			if s, _ := p.(string); !strings.HasPrefix(s, "#/") {
				t.Fatalf("nav item %v: match %v is not a #/ hash", item, p)
			}
		}
		hrefs = append(hrefs, href)
	}
	if got := strings.Join(hrefs, " "); got != "#/ #/map #/history" || len(items) > 8 {
		t.Fatalf("nav.items hrefs %q", got)
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
