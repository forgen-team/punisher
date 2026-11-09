package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wooo-jin/punisher/internal/learning"
)

func TestRSSSignalsClientAndApply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/internal/users/u%201/learning-signals" && r.URL.Path != "/v1/internal/users/u 1/learning-signals" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("X-Internal-Token") != "tok" || r.Header.Get("X-Internal-User-Id") != "u 1" {
			t.Errorf("auth headers missing")
		}
		_, _ = w.Write([]byte(`{"interests":["RAG"],"trending":[{"topic":"eBPF","z":3.5}],"explanationConcepts":[{"conceptId":"os.process","count":4}]}`))
	}))
	defer srv.Close()
	sig, err := RSSSignals{BaseURL: srv.URL + "/", Token: "tok"}.LearningSignals(context.Background(), "u 1")
	if err != nil {
		t.Fatal(err)
	}
	in := learning.SuggestionInput{Deepen: []learning.DeepenSignal{{ConceptID: "os.process", Name: "프로세스", Remaining: 2}}}
	applySignals(&in, sig)
	if len(in.Interests) != 1 || len(in.Trending) != 1 || in.Trending[0].Score != 3.5 || in.Deepen[0].Mentions != 4 {
		t.Fatalf("applied = %+v", in)
	}

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer failing.Close()
	if _, err := (RSSSignals{BaseURL: failing.URL, Token: "x"}).LearningSignals(context.Background(), "u"); err == nil {
		t.Fatal("non-200 must be an error (never treated as empty success)")
	}
}
