package learning

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wooo-jin/punisher/internal/lecture"
)

func blocks(bs ...lecture.Block) lecture.Content {
	return lecture.Content{Sections: []lecture.Section{{Kind: lecture.SectionPrinciples, Blocks: bs}}}
}

func TestEstimateMinutes(t *testing.T) {
	trace := `{"caption":"","columns":["a","b"],"rows":[` + strings.TrimSuffix(strings.Repeat(`["1","2"],`, 10), ",") + `]}`
	cases := []struct {
		name string
		c    lecture.Content
		want int
	}{
		{"empty is at least 1", lecture.Content{}, 1},
		// 6,000 prose + 2,000 code (the critiqued lectures): 10 + 4 = 14.
		{"legacy lecture", blocks(
			lecture.Block{Type: lecture.BlockMarkdown, Text: strings.Repeat("가", 6000)},
			lecture.Block{Type: lecture.BlockCode, Text: strings.Repeat("x", 2000)}), 14},
		// spaces and newlines are not counted.
		{"whitespace ignored", blocks(lecture.Block{Type: lecture.BlockMarkdown, Text: strings.Repeat("가 \n", 1200)}), 2},
		// 8,500/600 + 2,500/500 + 2×1.5 + 1×0.5 + 10×0.1 = 14.17 + 5 + 3 + 0.5 + 1 = 23.67 → 24.
		{"design target", blocks(
			lecture.Block{Type: lecture.BlockMarkdown, Text: strings.Repeat("가", 8500)},
			lecture.Block{Type: lecture.BlockLog, Text: strings.Repeat("x", 2500)},
			lecture.Block{Type: lecture.BlockExercise}, lecture.Block{Type: lecture.BlockAnswer},
			lecture.Block{Type: lecture.BlockExercise}, lecture.Block{Type: lecture.BlockAnswer},
			lecture.Block{Type: lecture.BlockPause},
			lecture.Block{Type: lecture.BlockTrace, Data: []byte(trace)}), 24},
		// rounding: 900/600 = 1.5 → 2; 840/600 = 1.4 → 1.
		{"round half up", blocks(lecture.Block{Type: lecture.BlockMarkdown, Text: strings.Repeat("가", 900)}), 2},
		{"round down", blocks(lecture.Block{Type: lecture.BlockMarkdown, Text: strings.Repeat("가", 840)}), 1},
		// code fenced inside markdown reads at the code rate: 1,000/500 = 2.
		{"fenced code", blocks(lecture.Block{Type: lecture.BlockMarkdown, Text: "```\n" + strings.Repeat("x", 1000) + "\n```"}), 2},
	}
	for _, tc := range cases {
		if got := EstimateMinutes(tc.c); got != tc.want {
			t.Errorf("%s: EstimateMinutes = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestSectionMinutes(t *testing.T) {
	c := lecture.Content{Sections: []lecture.Section{
		{Anchor: "a", Minutes: 9, Blocks: []lecture.Block{{Type: lecture.BlockMarkdown, Text: strings.Repeat("가", 900)}}}, // 1.5 → 2
		{Anchor: "b", Minutes: 9, Blocks: []lecture.Block{{Type: lecture.BlockMarkdown, Text: strings.Repeat("가", 900)}}}, // 1.5 → 2
	}}
	got := SectionMinutes(c)
	// 각자 반올림(2+2)은 전체 계산(1800/600=3)과 1분 어긋날 수 있다.
	if len(got) != 2 || got[0] != 2 || got[1] != 2 || EstimateMinutes(c) != 3 {
		t.Fatalf("SectionMinutes = %v, total %d", got, EstimateMinutes(c))
	}
	raw := []byte(`{"title":"t","extra":true,"sections":[{"anchor":"a","minutes":9,"blocks":[{"type":"markdown","text":"` +
		strings.Repeat("가", 1200) + `"}],"future":1}]}`)
	var out struct {
		Extra    bool `json:"extra"`
		Sections []struct {
			Minutes          int `json:"minutes"`
			EstimatedMinutes int `json:"estimatedMinutes"`
			Future           int `json:"future"`
		} `json:"sections"`
	}
	if err := json.Unmarshal(WithSectionMinutes(raw), &out); err != nil {
		t.Fatal(err)
	}
	if !out.Extra || len(out.Sections) != 1 || out.Sections[0].EstimatedMinutes != 2 || out.Sections[0].Minutes != 9 || out.Sections[0].Future != 1 {
		t.Fatalf("WithSectionMinutes = %+v", out)
	}
	if bad := []byte(`not json`); string(WithSectionMinutes(bad)) != "not json" {
		t.Fatal("undecodable content must pass through")
	}
}

func TestSessionFor(t *testing.T) {
	review := ReviewToday{EstimatedMinutes: 6}
	if s := SessionFor([]Recommendation{{EstimatedMinutes: 18}, {EstimatedMinutes: 20}}, review); s != (SessionMinutes{LectureMinutes: 18, CheckMinutes: 7, ReviewMinutes: 6}) {
		t.Fatalf("session = %+v", s)
	}
	if s := SessionFor(nil, review); s != (SessionMinutes{ReviewMinutes: 6}) {
		t.Fatalf("nothing to learn: %+v", s)
	}
	res := BuildToday(TodayInput{Concepts: map[string]TodayConcept{"a": {ID: "a", Depth: 1, HasPending: true, PendingDepths: 1}},
		Resume: []ResumeItem{{LectureID: "l", ConceptID: "a", Percent: 50, EstimatedMinutes: 14}}})
	if res.Session.LectureMinutes != 7 || res.Session.CheckMinutes != 7 || res.Review.LectureMinutes != 7 {
		t.Fatalf("resume: half of 14 minutes left = %+v / review %+v", res.Session, res.Review)
	}
}
