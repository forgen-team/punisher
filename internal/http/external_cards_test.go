package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgen-team/punisher/internal/store"
	"github.com/forgen-team/punisher/internal/witems"
	"github.com/forgen-team/punisher/migrations"
)

func newCardRouter(t *testing.T, guard func(http.Handler) http.Handler) http.Handler {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "punisher.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background(), migrations.Files); err != nil {
		t.Fatal(err)
	}
	return NewRouter(RouterDependencies{Store: st, Guard: guard})
}

// send makes a request as a server-to-server caller would (no Origin, no proxy headers).
func send(t *testing.T, h http.Handler, method, path, body string, headers map[string]string) (int, store.ReviewCard) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = "127.0.0.1:4710"
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var c store.ReviewCard
	if rec.Code == http.StatusOK && method != http.MethodGet {
		if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
			t.Fatalf("%s %s: %v (%s)", method, path, err, rec.Body.String())
		}
	}
	return rec.Code, c
}

func listCards(t *testing.T, h http.Handler, headers map[string]string) []store.ReviewCard {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/cards", nil)
	req.Host = "127.0.0.1:4710"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out struct {
		Cards []store.ReviewCard `json:"cards"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &out) != nil {
		t.Fatalf("list cards: %d %s", rec.Code, rec.Body.String())
	}
	return out.Cards
}

const externalBody = `{"sourceKind":"external","externalId":"atlas:01HCARD","sourceUrl":"http://127.0.0.1:4720/#/cards?card=01HCARD",
	"front":"수납: 본인부담 상한을 넘으면?","back":"상한을 넘은 금액은 환급한다.\n근거: payment/copay.ts:10-14"}`

// TestExternalCardsStandalone: the standalone app's loopback rule admits a
// local server call; POST twice keeps one card; PATCH by external id or by id
// sets and clears stale; another page's Origin is refused.
func TestExternalCardsStandalone(t *testing.T) {
	h := newCardRouter(t, nil)
	code, first := send(t, h, http.MethodPost, "/v1/cards", externalBody, nil)
	if code != http.StatusOK || first.SourceKind != "external" || first.ExternalID != "atlas:01HCARD" || first.Stale {
		t.Fatalf("post = %d %+v", code, first)
	}
	if code, again := send(t, h, http.MethodPost, "/v1/cards", externalBody, nil); code != http.StatusOK || again.ID != first.ID {
		t.Fatalf("second post = %d %+v", code, again)
	}
	if n := len(listCards(t, h, nil)); n != 1 {
		t.Fatalf("cards after two posts = %d", n)
	}
	code, p := send(t, h, http.MethodPatch, "/v1/cards/external/atlas%3A01HCARD", `{"stale":true}`, nil)
	if code != http.StatusOK || !p.Stale || p.Question != "수납: 본인부담 상한을 넘으면?" {
		t.Fatalf("patch stale = %d %+v", code, p)
	}
	if got := listCards(t, h, nil); !got[0].Stale || got[0].SourceURL == "" {
		t.Fatalf("listed = %+v", got[0])
	}
	if code, p = send(t, h, http.MethodPatch, "/v1/cards/"+first.ID, `{"stale":false,"back":"새 답"}`, nil); code != http.StatusOK || p.Stale || p.Answer != "새 답" {
		t.Fatalf("patch by id = %d %+v", code, p)
	}
	if code, _ := send(t, h, http.MethodPatch, "/v1/cards/external/atlas:none", `{"stale":true}`, nil); code != http.StatusNotFound {
		t.Fatalf("missing external id = %d", code)
	}
	if code, _ := send(t, h, http.MethodPost, "/v1/cards", `{"kind":"external","externalId":"atlas:x","sourceUrl":"javascript:1","front":"q","back":"a"}`, nil); code != http.StatusBadRequest {
		t.Fatalf("bad source url = %d", code)
	}
	if code, _ := send(t, h, http.MethodPost, "/v1/cards", externalBody, map[string]string{"Origin": "https://evil.example"}); code != http.StatusForbidden {
		t.Fatalf("foreign origin = %d", code)
	}
	if code, _ := send(t, h, http.MethodPatch, "/v1/cards/"+first.ID, `{"stale":true}`, map[string]string{"X-Forwarded-For": "100.64.0.2"}); code != http.StatusForbidden {
		t.Fatalf("proxied call = %d", code)
	}
}

// TestExternalCardsWitemsService: in the Witems app mode another service
// (atlas) calls with X-Internal-Token + X-Internal-User-Id of the owner; a
// wrong token, a non-owner or no token at all is refused.
func TestExternalCardsWitemsService(t *testing.T) {
	g := &witems.Guard{AuthServiceURL: "http://127.0.0.1:1", OwnerUserID: "owner-id", Origin: "https://witems.kr", InternalToken: "secret"}
	h := newCardRouter(t, g.Middleware)
	owner := map[string]string{"X-Internal-Token": "secret", "X-Internal-User-Id": "owner-id"}
	code, c := send(t, h, http.MethodPost, "/v1/cards", externalBody, owner)
	if code != http.StatusOK || c.ExternalID != "atlas:01HCARD" {
		t.Fatalf("service post = %d %+v", code, c)
	}
	if code, p := send(t, h, http.MethodPatch, "/v1/cards/external/atlas:01HCARD", `{"stale":true}`, owner); code != http.StatusOK || !p.Stale {
		t.Fatalf("service patch = %d %+v", code, p)
	}
	for name, hdr := range map[string]map[string]string{
		"wrong token": {"X-Internal-Token": "nope", "X-Internal-User-Id": "owner-id"},
		"no token":    {"X-Internal-User-Id": "owner-id"},
		"not owner":   {"X-Internal-Token": "secret", "X-Internal-User-Id": "other-id"},
		"no session":  {},
	} {
		want := http.StatusUnauthorized
		if name == "not owner" {
			want = http.StatusForbidden
		}
		if code, _ := send(t, h, http.MethodPatch, "/v1/cards/external/atlas:01HCARD", `{"stale":false}`, hdr); code != want {
			t.Errorf("%s: status %d, want %d", name, code, want)
		}
	}
	if got := listCards(t, h, owner); len(got) != 1 || !got[0].Stale {
		t.Fatalf("refused calls changed the card: %+v", got)
	}
}
