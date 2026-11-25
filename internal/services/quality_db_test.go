package services

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/forgen-team/punisher/internal/lecture"
)

// TestInstantQualityPipeline: with InstantOutline the draft gets the short
// plan; with LectureVerify the review gets the independent answers; the
// review row stores the rubric and only the quoted reviewer scores.
func TestInstantQualityPipeline(t *testing.T) {
	review := `{"verdict":"pass","summary":"좋음","issues":[],"rubric":[
		{"id":"E04","score":3,"quote":"**부호 없는 이진수**는 각 자리의","why":"정의 있음"},
		{"id":"E05","score":4,"quote":"본문에 없는 문장입니다","why":"x"},
		{"id":"E99","score":2,"quote":"부호 없는 이진수","why":"x"}]}`
	fake := &pipelineLLM{reviews: []string{review}, solved: `{"answers":[{"id":"principles-3","answer":"1001"}]}`}
	r := runInstantPrepared(t, fake, func(ids []string) []string {
		return []string{`{"thesis":"고정 폭 정수는 넘친다","key_definitions":[],"misconceptions":[],"scene_numbers":["2147483647"]}`, lectureMarkupFor(ids)}
	}, func(s *Service) { s.Cfg.InstantOutline, s.Cfg.LectureVerify = true, true })

	if got := r.stages(); len(got) == 0 || got[0] != "planning/0" || got[len(got)-1] != "published/0" {
		t.Fatalf("stages = %v", got)
	}
	if len(fake.draftUsers) != 2 || !strings.Contains(fake.draftUsers[1], "확정된 개요") || !strings.Contains(fake.draftUsers[1], "고정 폭 정수는 넘친다") {
		t.Fatalf("the draft gets the short plan: %d calls", len(fake.draftUsers))
	}
	if len(fake.solveUsers) != 1 || strings.Contains(fake.solveUsers[0], "1001 입니다") {
		t.Fatalf("the independent pass never sees the draft's answer: %q", fake.solveUsers)
	}
	if !strings.Contains(fake.reviewUsers[0], "따로 푼 답: 1001 / 본문: 1001 입니다.") || !strings.Contains(fake.reviewUsers[0], `"rubric"`) {
		t.Fatalf("review prompt lacks evidence or the rubric request")
	}

	var raw string
	if err := testDB(r.s).QueryRowContext(context.Background(), `SELECT r.quality FROM lecture_reviews r
		JOIN lectures l ON l.id = r.lecture_id WHERE l.generation_job_id = ? AND l.status = 'published'`, r.jobID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var q LectureQuality
	if err := json.Unmarshal([]byte(raw), &q); err != nil {
		t.Fatal(err)
	}
	if len(q.LLM) != 1 || q.LLM[0].ID != "E04" || len(q.Independent) != 1 || q.Machine.Rule("G23").Status != "fail" || q.PromptVersion == "" {
		t.Fatalf("stored quality = %s", raw)
	}
}

func TestValidRubric(t *testing.T) {
	markup := "@@markdown\n첫 문장입니다.\n둘째   문장은 줄이\n바뀝니다."
	got := validRubric([]RubricScore{
		{ID: "E01", Score: 3, Quote: "둘째 문장은 줄이 바뀝니다"},
		{ID: "E01", Score: 1, Quote: "첫 문장입니다"},
		{ID: "E02", Score: 5, Quote: "첫 문장입니다"},
		{ID: "E03", Score: 2, Quote: ""},
	}, markup)
	if len(got) != 1 || got[0].Score != 3 {
		t.Fatalf("valid rubric = %+v", got)
	}
}

func TestRevisionNotesCarryQualityNotes(t *testing.T) {
	notes := []lecture.Issue{{Path: "quality.G23", Message: "회상 프롬프트: principles 회상 0개", Advisory: true}}
	got := revisionNotes(Attempt{Reviewed: true, Review: ReviewOutcome{Verdict: "pass"}}, true, notes)
	if !strings.Contains(got, "구성 메모") || !strings.Contains(got, "회상 0개") {
		t.Fatalf("notes = %q", got)
	}
	if strings.Contains(revisionNotes(Attempt{}, false, notes), "구성 메모") {
		t.Fatal("the last chance fixes blocking findings only")
	}
}
