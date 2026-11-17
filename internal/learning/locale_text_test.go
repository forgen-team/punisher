package learning

import (
	"strings"
	"testing"
)

func TestTodayReasonsFollowLocale(t *testing.T) {
	in := TodayInput{
		Concepts: map[string]TodayConcept{"a": {ID: "a", Name: "A", Area: "os", Depth: 1, State: NotStarted, HasPending: true}},
		Resume:   []ResumeItem{{LectureID: "l", ConceptID: "a", Title: "T", Percent: 40}},
	}
	if r := BuildToday(in).Items[0].Reason; !strings.Contains(r, "지난번 40%") {
		t.Fatalf("ko reason: %q", r)
	}
	in.Locale = LocaleEN
	if r := BuildToday(in).Items[0].Reason; !strings.HasPrefix(r, "You read 40%") {
		t.Fatalf("en reason: %q", r)
	}
}

func TestSuggestionsFollowLocale(t *testing.T) {
	in := SuggestionInput{Deepen: []DeepenSignal{{ConceptID: "a", Name: "Cache", Remaining: 2}}, NewAreas: DefaultNewAreas, Locale: LocaleEN}
	for _, s := range RankSuggestions(in, 10) {
		for _, r := range s.Title + s.Reason {
			if r >= '가' && r <= '힣' {
				t.Fatalf("Korean in an en suggestion: %+v", s)
			}
		}
	}
}
