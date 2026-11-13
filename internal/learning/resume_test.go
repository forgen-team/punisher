package learning

import "testing"

func TestReplaceRegenerated(t *testing.T) {
	resume := []ResumeItem{
		{LectureID: "old", ConceptID: "c1", Title: "옛 강의", Anchor: "s1-p3", Percent: 8, ParentLectureID: "p"},
		{LectureID: "other", ConceptID: "c2", Title: "다른 강의", Percent: 40},
		{LectureID: "done-old", ConceptID: "c3", Title: "이미 새 버전을 읽음", Percent: 20},
	}
	newer := map[string]NewerVersion{
		"old":      {Item: ResumeItem{LectureID: "new", ConceptID: "c1", Title: "새 강의", EstimatedMinutes: 26}},
		"done-old": {Item: ResumeItem{LectureID: "done-new", ConceptID: "c3"}, Started: true},
	}
	got := ReplaceRegenerated(resume, newer)
	if len(got) != 2 {
		t.Fatalf("got %d items, want 2: %+v", len(got), got)
	}
	n := got[0]
	if n.LectureID != "new" || n.Title != "새 강의" || n.Percent != 0 || n.Anchor != "" || n.EstimatedMinutes != 26 || n.ParentLectureID != "p" {
		t.Fatalf("replacement = %+v", n)
	}
	if got[1].LectureID != "other" || got[1].Percent != 40 {
		t.Fatalf("untouched row changed: %+v", got[1])
	}
}

func TestReplaceRegeneratedFollowsChain(t *testing.T) {
	newer := map[string]NewerVersion{
		"v1": {Item: ResumeItem{LectureID: "v2"}},
		"v2": {Item: ResumeItem{LectureID: "v3"}},
	}
	got := ReplaceRegenerated([]ResumeItem{{LectureID: "v1", Percent: 30}}, newer)
	if len(got) != 1 || got[0].LectureID != "v3" {
		t.Fatalf("got %+v, want v3", got)
	}
	// A cycle must not hang.
	cyc := map[string]NewerVersion{"a": {Item: ResumeItem{LectureID: "b"}}, "b": {Item: ResumeItem{LectureID: "a"}}}
	_ = ReplaceRegenerated([]ResumeItem{{LectureID: "a"}}, cyc)
}
