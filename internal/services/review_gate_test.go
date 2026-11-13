package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wooo-jin/punisher/internal/llm"
)

var (
	advisoryOnly = ReviewOutcome{Verdict: "fail", Summary: "구성 부족", Issues: []ReviewIssue{
		{Severity: "major", Gate: GateAdvisory, Kind: "pedagogy", Section: "principles", Message: "연습이 부족"},
		{Severity: "major", Gate: GateBlocking, Kind: "structure", Message: "요약 불릿이 여러 개념"}, // advisory kind: never blocks
	}}
	blockingFact = ReviewOutcome{Verdict: "fail", Summary: "사실 오류", Issues: []ReviewIssue{
		{Severity: "major", Gate: GateBlocking, Kind: "factual_error", Section: "walkthrough", Quote: "ERROR:  integer out of range",
			Message: "출력이 실제와 다름", Suggestion: "출력 예시를 빼거나 조건을 붙임",
			Confidence: 0.9, ReaderImpact: "다른 DB 에서도 같은 에러가 난다고 믿는다"},
	}}
	conditionalClaim = ReviewOutcome{Verdict: "pass", Issues: []ReviewIssue{
		{Severity: "minor", Gate: GateAdvisory, Kind: "factual_error", Message: "확인 필요(조건이 붙어 있음)"},
	}}
	cleanPass = ReviewOutcome{Verdict: "pass", Issues: []ReviewIssue{}}
)

func TestReviewGates(t *testing.T) {
	cases := []struct {
		name string
		r    ReviewOutcome
		want bool
	}{
		{"advisory only, even with verdict fail", advisoryOnly, false},
		{"blocking factual error", blockingFact, true},
		{"conditional unverifiable claim is advisory", conditionalClaim, false},
		{"legacy review without gate: major factual blocks", ReviewOutcome{Verdict: "pass", Issues: []ReviewIssue{{Severity: "major", Kind: "factual_error", Confidence: 0.8, ReaderImpact: "r"}}}, true},
		{"legacy review without gate: major pedagogy does not", ReviewOutcome{Verdict: "fail", Issues: []ReviewIssue{{Severity: "major", Kind: "pedagogy"}}}, false},
		{"fail without any finding blocks", ReviewOutcome{Verdict: "fail", Issues: []ReviewIssue{}}, true},
		{"unknown kind keeps the severity rule", ReviewOutcome{Verdict: "pass", Issues: []ReviewIssue{{Severity: "critical", Confidence: 0.8, ReaderImpact: "r"}}}, true},
	}
	for _, tc := range cases {
		if got := tc.r.Blocking(); got != tc.want {
			t.Errorf("%s: Blocking() = %v, want %v", tc.name, got, tc.want)
		}
	}
	if r := ParseReview(`{"verdict":"fail","issues":[{"severity":"major","gate":"advisory","kind":"overclaim","message":"m"}]}`); r.Blocking() || r.Issues[0].Gate != GateAdvisory {
		t.Fatalf("gate must be parsed: %+v", r)
	}
}

// advisory only → published at once when no attempt is left, or after one
// advisory rewrite that also passes.
func TestReviewLoopPublishesAdvisoryOnly(t *testing.T) {
	f := &fakeSteps{drafts: []string{validMarkup}, reviews: []ReviewOutcome{advisoryOnly}}
	res, err := RunReviewLoop(context.Background(), 1, f.funcs())
	if err != nil || res.Status != "published" || res.Final != 0 {
		t.Fatalf("advisory-only draft must publish: res=%+v err=%v", res, err)
	}
	log := res.AttemptLog()
	if log[0].Result != "ok" || !log[0].Published || len(log[0].Advisory) != 2 || !strings.Contains(log[0].Advisory[0], "연습이 부족") {
		t.Fatalf("advisory findings must stay in the attempt log: %+v", log)
	}

	f = &fakeSteps{drafts: []string{validMarkup, validMarkup}, reviews: []ReviewOutcome{advisoryOnly, cleanPass}}
	res, err = RunReviewLoop(context.Background(), 3, f.funcs())
	if err != nil || res.Status != "published" || res.Final != 1 || len(res.Attempts) != 2 {
		t.Fatalf("one advisory rewrite, then publish: res=%+v err=%v", res, err)
	}
	if !strings.Contains(f.notes[1], "### 개선 권장") || !strings.Contains(f.notes[1], "연습이 부족") || f.prev[1] != validMarkup {
		t.Fatalf("the advisory rewrite gets the findings and the draft to edit: %q", f.notes[1])
	}
}

// Only one advisory rewrite: a second advisory-only pass is published.
func TestReviewLoopOnlyOneAdvisoryRewrite(t *testing.T) {
	f := &fakeSteps{drafts: []string{validMarkup, validMarkup, validMarkup}, reviews: []ReviewOutcome{advisoryOnly, advisoryOnly, cleanPass}}
	res, err := RunReviewLoop(context.Background(), 3, f.funcs())
	if err != nil || res.Status != "published" || res.Final != 1 || f.gen != 2 {
		t.Fatalf("res=%+v err=%v drafts=%d", res, err, f.gen)
	}
}

// blocking → rewrite → blocking resolved → published.
func TestReviewLoopBlockingResolvedByRewrite(t *testing.T) {
	f := &fakeSteps{drafts: []string{validMarkup, validMarkup}, reviews: []ReviewOutcome{blockingFact, conditionalClaim}}
	res, err := RunReviewLoop(context.Background(), 3, f.funcs())
	if err != nil || res.Status != "published" || res.Final != 1 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if !strings.Contains(f.notes[1], "### 반드시 고칠 것") || !strings.Contains(f.notes[1], "출력이 실제와 다름") || f.prev[1] != validMarkup {
		t.Fatalf("the rewrite edits the previous draft with the blocking finding: %q", f.notes[1])
	}
	log := res.AttemptLog()
	if log[0].Result != "review_failed" || !strings.Contains(log[0].Message, "출력이 실제와 다름") || log[0].Published || !log[1].Published {
		t.Fatalf("attempt log = %+v", log)
	}
}

// The advisory rewrite introduces a blocking finding → the earlier passing
// draft is published.
func TestReviewLoopKeepsPassingDraftWhenRewriteBreaks(t *testing.T) {
	f := &fakeSteps{drafts: []string{validMarkup, validMarkup}, reviews: []ReviewOutcome{advisoryOnly, blockingFact}}
	res, err := RunReviewLoop(context.Background(), 3, f.funcs())
	if err != nil || res.Status != "published" || res.Final != 0 || len(res.Attempts) != 2 {
		t.Fatalf("earlier passing draft must be published: res=%+v err=%v", res, err)
	}
	if f.gen != 2 {
		t.Fatalf("no further rewrite after the broken one, drafts = %d", f.gen)
	}
	if log := res.AttemptLog(); !log[0].Published || log[1].Published || log[1].Result != "review_failed" {
		t.Fatalf("attempt log = %+v", log)
	}
}

// Machine composition / volume findings are advisory: reviewed and published.
func TestReviewLoopMachineAdvisoryDoesNotBlock(t *testing.T) {
	noPause := strings.Replace(validMarkup, "@@pause\n위 쿼리는 무엇을 출력할까요?\n", "", 1)
	f := &fakeSteps{drafts: []string{noPause}, reviews: []ReviewOutcome{cleanPass}}
	loop := f.funcs()
	loop.SkipReviewOnIssues = true
	res, err := RunReviewLoop(context.Background(), 1, loop)
	if err != nil || res.Status != "published" || f.rev != 1 {
		t.Fatalf("res=%+v err=%v reviews=%d", res, err, f.rev)
	}
	if log := res.AttemptLog(); !strings.Contains(strings.Join(log[0].Advisory, " "), "@@pause") {
		t.Fatalf("machine advisory must be logged: %+v", log)
	}
	// A self link still blocks (and skips the review).
	selfLink := strings.Replace(validMarkup, "**부호 없는 이진수**는", "[개념](term:c)은", 1)
	f = &fakeSteps{drafts: []string{selfLink}, reviews: []ReviewOutcome{cleanPass}}
	loop = f.funcs()
	loop.SkipReviewOnIssues = true
	if res, err := RunReviewLoop(context.Background(), 1, loop); err != nil || res.Status != "needs_revision" || f.rev != 0 {
		t.Fatalf("self link must block: res=%+v err=%v", res, err)
	}
}

// A draft or review over its attempt limit fails that attempt only.
func TestReviewLoopAttemptTimeout(t *testing.T) {
	f := &fakeSteps{drafts: []string{validMarkup}, reviews: []ReviewOutcome{cleanPass}, hang: map[int]bool{0: true}}
	loop := f.funcs()
	loop.DraftTimeout = 30 * time.Millisecond
	res, err := RunReviewLoop(context.Background(), 2, loop)
	if err != nil || res.Status != "published" || res.Final != 1 {
		t.Fatalf("the next attempt must run after a timed-out draft: res=%+v err=%v", res, err)
	}
	log := res.AttemptLog()
	if log[0].Result != "timeout" || log[0].DurationMs < 30 || log[1].Result != "ok" {
		t.Fatalf("attempt log = %+v", log)
	}

	// A review over its limit after a passing draft: the passing draft is published.
	f = &fakeSteps{drafts: []string{validMarkup, validMarkup}, reviews: []ReviewOutcome{advisoryOnly}, reviewHang: map[int]bool{1: true}}
	loop = f.funcs()
	loop.ReviewTimeout = 30 * time.Millisecond
	res, err = RunReviewLoop(context.Background(), 3, loop)
	if err != nil || res.Status != "published" || res.Final != 0 || res.AttemptLog()[1].Result != "timeout" {
		t.Fatalf("res=%+v err=%v", res, err)
	}

	// Every attempt timed out: the job fails with the timeout.
	f = &fakeSteps{drafts: []string{validMarkup}, hang: map[int]bool{0: true, 1: true}}
	loop = f.funcs()
	loop.DraftTimeout = 20 * time.Millisecond
	if _, err := RunReviewLoop(context.Background(), 2, loop); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want the timeout, got %v", err)
	}

	// A provider timeout (gateway queue/run limit) counts as an attempt timeout too.
	if !isAttemptTimeout(&llm.LLMError{Err: llm.ErrTimeout}) || isAttemptTimeout(errors.New("429")) {
		t.Fatal("isAttemptTimeout")
	}
}

// A partial rewrite is compared block by block with the draft it revised.
func TestReviewLoopMeasuresRewriteDrift(t *testing.T) {
	// The finding quotes the walkthrough log; the rewrite changes only that block.
	fixed := strings.Replace(validMarkup, "ERROR:  integer out of range", "(출력은 DB 와 버전에 따라 다름)", 1)
	f := &fakeSteps{drafts: []string{validMarkup, fixed}, reviews: []ReviewOutcome{blockingFact, cleanPass}}
	res, err := RunReviewLoop(context.Background(), 3, f.funcs())
	if err != nil || res.Status != "published" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	d := res.Attempts[1].Drift
	if d == nil || d.Changed != 0 || d.Unrelated == 0 || d.Warn() {
		t.Fatalf("an in-place fix must not drift: %+v", d)
	}

	// A rewrite that rewrites unrelated prose is flagged.
	rewritten := strings.ReplaceAll(validMarkup, "정수는 고정된 비트 폭 안에서만 값을 담습니다.", "고정 폭 정수는 담을 수 있는 범위가 정해져 있습니다.")
	rewritten = strings.Replace(rewritten, "0110 + 0011 의 4비트 결과는?", "0101 + 0011 의 결과는?", 1)
	rewritten = strings.Replace(rewritten, "1001 입니다.", "1000 입니다.", 1)
	f = &fakeSteps{drafts: []string{validMarkup, rewritten}, reviews: []ReviewOutcome{blockingFact, cleanPass}}
	res, err = RunReviewLoop(context.Background(), 3, f.funcs())
	if err != nil {
		t.Fatal(err)
	}
	d = res.Attempts[1].Drift
	if d == nil || !d.Warn() || !strings.HasPrefix(res.AttemptLog()[1].Drift, "warning:") {
		t.Fatalf("unrelated changes must be flagged: %+v", d)
	}
}
