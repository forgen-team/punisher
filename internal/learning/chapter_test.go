package learning

import "testing"

func TestRemainingMinutesAndTrigger(t *testing.T) {
	concepts := map[string]TodayConcept{
		"a": {ID: "a", PendingDepths: 2},                    // 60
		"b": {ID: "b", State: Completed, PendingDepths: 3},  // done: excluded
		"c": {ID: "c", Proposed: true, PendingDepths: 1},    // proposed: excluded
		"d": {ID: "d", State: InProgress, PendingDepths: 1}, // 30
	}
	if got := RemainingMinutes(concepts); got != 90 {
		t.Fatalf("remaining = %d", got)
	}
	if p := CheckNextChapter(90, 0); p == nil || p.ThresholdMinutes != 30*ChapterRunwayDays || p.DailyMinutes != 30 {
		t.Fatalf("default pace prompt = %+v", p)
	}
	if CheckNextChapter(420, 30) != nil {
		t.Fatal("remaining == threshold must not trigger")
	}
	if CheckNextChapter(419, 30) == nil || CheckNextChapter(800, 60) == nil || CheckNextChapter(800, 15) != nil {
		t.Fatal("threshold must scale with daily minutes")
	}
}

func TestRankSuggestions(t *testing.T) {
	in := SuggestionInput{
		Proposed: []ProposedSignal{{ID: "p1", Name: "RAG 파이프라인", Mentions: 3}, {ID: "p2", Name: "eBPF", Mentions: 1},
			{ID: "p3", Name: "WAL", Mentions: 0}},
		Deepen:    []DeepenSignal{{ConceptID: "os.process", Name: "프로세스", Remaining: 4, Mentions: 2}},
		Interests: []string{"RAG", "프로세스"},
		Trending:  []TrendSignal{{Topic: "Kubernetes", Score: 4}, {Topic: "프로세스", Score: 9}},
		NewAreas:  DefaultNewAreas,
		Covered:   map[string]bool{"프로세스": true},
	}
	got := RankSuggestions(in, 5)
	if len(got) != 5 {
		t.Fatalf("len = %d: %+v", len(got), got)
	}
	if got[0].ID != "proposed:p1" {
		t.Fatalf("top = %+v (proposed + interest match should win)", got[0])
	}
	kinds := map[string]int{}
	for _, s := range got {
		kinds[s.Kind]++
		if s.Reason == "" || s.Title == "" {
			t.Fatalf("suggestion without reason/title: %+v", s)
		}
		if s.Kind == SuggestTrend && s.Title == "프로세스" {
			t.Fatal("topics already on the map must not be suggested as trends")
		}
	}
	for k, n := range kinds {
		if n > maxPerKind {
			t.Fatalf("kind %s appears %d times", k, n)
		}
	}
	// LLM area gets the interest boost (RAG ~ rag keyword).
	var llm Suggestion
	for _, s := range RankSuggestions(in, 20) {
		if s.ID == "new_area:llm-engineering" {
			llm = s
		}
	}
	if llm.Score < 2.3 {
		t.Fatalf("llm area not boosted: %+v", llm)
	}
}

func TestRankSuggestionsFillsWhenFewSources(t *testing.T) {
	got := RankSuggestions(SuggestionInput{NewAreas: DefaultNewAreas}, 5)
	if len(got) != len(DefaultNewAreas) {
		t.Fatalf("with only new areas every one should be offered, got %d", len(got))
	}
}
