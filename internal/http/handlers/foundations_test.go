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
	appMiddleware "github.com/forgen-team/punisher/internal/middleware"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/forgen-team/punisher/migrations"
)

// TestFoundationsFlagByReaderLevel: GET /v1/concepts, /v1/graph and
// /v1/me/frontier mark (or drop) the foundations track by the caller's level —
// hidden for unset/practitioner/advanced, shown for beginner/novice — and a
// level change only changes the flag.
func TestFoundationsFlagByReaderLevel(t *testing.T) {
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
	h := &ConceptsHandler{Store: st}
	get := func(handler http.HandlerFunc, url string, out any) {
		t.Helper()
		rec := httptest.NewRecorder()
		handler(rec, httptest.NewRequest(http.MethodGet, url, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d: %s", url, rec.Code, rec.Body.String())
		}
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatal(err)
		}
	}
	type flagged struct {
		ID              string `json:"id"`
		Area            string `json:"area"`
		HiddenByDefault bool   `json:"hiddenByDefault"`
	}
	check := func(level string, wantHidden bool) {
		t.Helper()
		if level != "" {
			l := level
			if _, err := st.UpdateLearningSettings(ctx, appMiddleware.LocalUserID, store.LearningSettingsPatch{ReaderLevel: &l}); err != nil {
				t.Fatal(err)
			}
		}
		var list struct {
			Concepts []flagged `json:"concepts"`
		}
		get(h.List, "/v1/concepts", &list)
		var graph struct {
			Nodes             []flagged `json:"nodes"`
			FoundationsHidden bool      `json:"foundationsHidden"`
		}
		get(h.Graph, "/v1/graph", &graph)
		if graph.FoundationsHidden != wantHidden {
			t.Errorf("level %q: graph foundationsHidden = %v", level, graph.FoundationsHidden)
		}
		for name, items := range map[string][]flagged{"concepts": list.Concepts, "graph": graph.Nodes} {
			found := 0
			for _, c := range items {
				isFound := c.Area == curriculum.AreaFoundations
				if isFound {
					found++
				}
				if c.HiddenByDefault != (isFound && wantHidden) {
					t.Errorf("level %q %s: %s hiddenByDefault = %v", level, name, c.ID, c.HiddenByDefault)
				}
			}
			if found != len(curriculum.FoundationsTrack) {
				t.Errorf("level %q %s: %d foundations concepts returned (always listed, only flagged)", level, name, found)
			}
		}
		var frontier struct {
			Items []flagged `json:"items"`
		}
		get(h.Frontier, "/v1/me/frontier?limit=200", &frontier)
		sawFoundations := false
		for _, c := range frontier.Items {
			sawFoundations = sawFoundations || c.Area == curriculum.AreaFoundations
		}
		if sawFoundations == wantHidden {
			t.Errorf("level %q: frontier has foundations = %v", level, sawFoundations)
		}
	}
	check("", true)
	check(learning.ReaderPractitioner, true)
	check(learning.ReaderBeginner, false)
	check(learning.ReaderNovice, false)
	check(learning.ReaderAdvanced, true)
}
