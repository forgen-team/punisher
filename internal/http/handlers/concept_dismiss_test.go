package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/services"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/forgen-team/punisher/migrations"
	"github.com/go-chi/chi/v5"
)

func dismissRouter(t *testing.T) http.Handler {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "punisher.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx, migrations.Files); err != nil {
		t.Fatal(err)
	}
	seed, err := curriculum.Load(seedfiles.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.SyncCurriculum(ctx, seed); err != nil {
		t.Fatal(err)
	}
	learn := &LearnHandler{Store: st, Svc: &services.Service{Store: st, Keys: services.GatewayKeys{}, Hub: services.NewHub()}}
	concepts := &ConceptsHandler{Store: st}
	r := chi.NewRouter()
	r.Post("/v1/concepts/{id}/dismiss", learn.DismissConcept)
	r.Delete("/v1/concepts/{id}/dismiss", learn.RestoreConcept)
	r.Get("/v1/concepts/{id}", concepts.Detail)
	r.Get("/v1/graph", concepts.Graph)
	r.Get("/v1/me/frontier", concepts.Frontier)
	r.Get("/v1/today", learn.Today)
	return r
}

func call(t *testing.T, h http.Handler, method, url string, wantCode int, out any) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, url, nil))
	if rec.Code != wantCode {
		t.Fatalf("%s %s: status %d: %s", method, url, rec.Code, rec.Body.String())
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatal(err)
		}
	}
}

type dismissView struct {
	frontier map[string]bool
	today    map[string]bool
	node     map[string]struct{ dismissed, locked, frontier bool }
}

func loadDismissView(t *testing.T, h http.Handler) dismissView {
	t.Helper()
	var fr struct {
		Items []struct{ ID string } `json:"items"`
	}
	call(t, h, http.MethodGet, "/v1/me/frontier?limit=200", http.StatusOK, &fr)
	var td struct {
		Items []struct {
			ConceptID string `json:"conceptId"`
		} `json:"items"`
	}
	call(t, h, http.MethodGet, "/v1/today?limit=20", http.StatusOK, &td)
	var g struct {
		Nodes []struct {
			ID        string `json:"id"`
			Dismissed bool   `json:"dismissed"`
			Locked    bool   `json:"locked"`
			Frontier  bool   `json:"frontier"`
		} `json:"nodes"`
	}
	call(t, h, http.MethodGet, "/v1/graph", http.StatusOK, &g)
	v := dismissView{frontier: map[string]bool{}, today: map[string]bool{}, node: map[string]struct{ dismissed, locked, frontier bool }{}}
	for _, it := range fr.Items {
		v.frontier[it.ID] = true
	}
	for _, it := range td.Items {
		v.today[it.ConceptID] = true
	}
	for _, n := range g.Nodes {
		v.node[n.ID] = struct{ dismissed, locked, frontier bool }{n.Dismissed, n.Locked, n.Frontier}
	}
	return v
}

// TestDismissConceptEndpoints: POST/DELETE /v1/concepts/{id}/dismiss toggles
// '필요 없음' — the concept leaves the frontier and today's list, stays on the
// map dimmed (dismissed, not frontier), keeps its dependents locked, and comes
// back on undo. Unknown concepts are 404.
func TestDismissConceptEndpoints(t *testing.T) {
	h := dismissRouter(t)
	before := loadDismissView(t, h)
	var target string
	for id := range before.today {
		if before.frontier[id] {
			target = id
			break
		}
	}
	if target == "" {
		t.Fatal("no concept that is both in today's list and on the frontier")
	}

	var res struct {
		ConceptID string `json:"conceptId"`
		Dismissed bool   `json:"dismissed"`
	}
	call(t, h, http.MethodPost, "/v1/concepts/"+target+"/dismiss", http.StatusOK, &res)
	if res.ConceptID != target || !res.Dismissed {
		t.Fatalf("dismiss response %+v", res)
	}
	var detail struct {
		Dismissed  bool   `json:"dismissed"`
		UserState  string `json:"userState"`
		Dependents []struct {
			ID string `json:"id"`
		} `json:"dependents"`
	}
	call(t, h, http.MethodGet, "/v1/concepts/"+target, http.StatusOK, &detail)
	if !detail.Dismissed || detail.UserState != learning.NotStarted {
		t.Errorf("detail: dismissed=%v userState=%q (state must stay as is)", detail.Dismissed, detail.UserState)
	}
	after := loadDismissView(t, h)
	if after.frontier[target] || after.today[target] {
		t.Errorf("dismissed %s still recommended: frontier=%v today=%v", target, after.frontier[target], after.today[target])
	}
	if n := after.node[target]; !n.dismissed || n.frontier {
		t.Errorf("map node %s: %+v, want dismissed and not frontier", target, n)
	}
	for _, d := range detail.Dependents {
		if n, ok := after.node[d.ID]; ok && !n.locked {
			t.Errorf("dependent %s unlocked by a dismissed prerequisite", d.ID)
		}
	}

	call(t, h, http.MethodDelete, "/v1/concepts/"+target+"/dismiss", http.StatusOK, &res)
	if res.Dismissed {
		t.Fatalf("undo response %+v", res)
	}
	if undone := loadDismissView(t, h); !undone.frontier[target] || undone.node[target].dismissed {
		t.Errorf("undo: frontier=%v node=%+v", undone.frontier[target], undone.node[target])
	}
	call(t, h, http.MethodPost, "/v1/concepts/no.such-concept/dismiss", http.StatusNotFound, nil)
}
