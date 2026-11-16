package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/forgen-team/punisher/internal/services"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/forgen-team/punisher/migrations"
)

// TestTodayResponseKeys: GET /v1/today carries every TodayResult part the web
// reads — the service test calls s.Today directly and cannot see a key the
// handler leaves out of the payload (recheck was missing).
// 단독판: 임시 SQLite, 인증 컨텍스트 없이(GetUserID 는 고정 로컬 사용자).
func TestTodayResponseKeys(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "punisher.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx, migrations.Files); err != nil {
		t.Fatal(err)
	}
	h := &LearnHandler{Store: st, Svc: &services.Service{Store: st, Keys: services.GatewayKeys{}, Hub: services.NewHub()}}

	req := httptest.NewRequest(http.MethodGet, "/v1/today?limit=5", nil)
	rec := httptest.NewRecorder()
	h.Today(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"items", "exhausted", "review", "recheck"} {
		if _, ok := body[key]; !ok {
			t.Errorf("today response has no %q: %s", key, rec.Body.String())
		}
	}
	var recheck []any
	if err := json.Unmarshal(body["recheck"], &recheck); err != nil || recheck == nil {
		t.Fatalf("recheck must be an array (empty when nothing lapsed): %s (%v)", body["recheck"], err)
	}
}
