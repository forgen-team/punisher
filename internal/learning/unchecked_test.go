package learning

import "testing"

func TestUncheckedCards(t *testing.T) {
	in := TodayInput{
		Concepts: map[string]TodayConcept{
			"a": {ID: "a", Name: "A", Area: "os"}, "b": {ID: "b", Name: "B"}, "c": {ID: "c", Name: "C"}, "d": {ID: "d", Name: "D"},
		},
		Unchecked: []UncheckedLecture{
			{LectureID: "la2", ConceptID: "a", Title: "A 최근"}, {LectureID: "la1", ConceptID: "a"},
			{LectureID: "lx", ConceptID: "gone"}, {LectureID: "lb", ConceptID: "b"}, {LectureID: "lc", ConceptID: "c"}, {LectureID: "ld", ConceptID: "d"},
		},
	}
	got := BuildToday(in).Unchecked
	if len(got) != MaxUncheckedCards || got[0].LectureID != "la2" || got[0].ConceptName != "A" || got[0].Area != "os" || got[1].LectureID != "lb" || got[2].LectureID != "lc" {
		t.Fatalf("cards = %+v", got)
	}
	if empty := BuildToday(TodayInput{}).Unchecked; empty == nil || len(empty) != 0 {
		t.Fatalf("no input must give an empty list, got %#v", empty)
	}
}

func TestLatestVersions(t *testing.T) {
	newer := map[string]NewerVersion{
		"v1": {Item: ResumeItem{LectureID: "v2", ConceptID: "c", Title: "2판"}}, "v2": {Item: ResumeItem{LectureID: "v3", ConceptID: "c", Title: "3판"}},
		"y": {Item: ResumeItem{LectureID: "other", ConceptID: "d"}},
	}
	got := LatestVersions([]UncheckedLecture{{LectureID: "v1", ConceptID: "c", Title: "1판"}, {LectureID: "x", Title: "그대로"}, {LectureID: "y", ConceptID: "c"}}, newer)
	if got[0].LectureID != "v3" || got[0].Title != "3판" || got[1].LectureID != "x" || got[2].LectureID != "y" {
		t.Fatalf("got %+v", got)
	}
}
