package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/wooo-jin/punisher/internal/lecture"
)

// convergeFake adds a scripted targeted recheck and adjudication to fakeSteps.
type convergeFake struct {
	*fakeSteps
	mu          sync.Mutex
	rechecks    []RecheckOutcome
	recheckIn   []RecheckInput
	decisions   map[string]string // finding message → uphold | dismiss
	adjudicated []string
}

func (c *convergeFake) funcs() LoopFuncs {
	f := c.fakeSteps.funcs()
	f.NoAdvisoryRewrite = true
	f.Recheck = func(_ context.Context, in RecheckInput) (RecheckOutcome, error) {
		c.recheckIn = append(c.recheckIn, in)
		if len(c.rechecks) == 0 {
			return RecheckOutcome{}, errors.New("no scripted recheck")
		}
		out := c.rechecks[0]
		c.rechecks = c.rechecks[1:]
		return out, nil
	}
	f.Adjudicate = func(_ context.Context, in AdjudicationInput) (AdjudicationResult, error) {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.adjudicated = append(c.adjudicated, in.Issue.Message)
		d := c.decisions[in.Issue.Message]
		if d == "" {
			d = AdjudicationUphold
		}
		return AdjudicationResult{Decision: d, Reason: "재판정 " + d}, nil
	}
	return f
}

const fillerSentence = "정수는 고정된 비트 폭 안에서만 값을 담습니다."

var fixedLog = strings.Replace(validMarkup, "ERROR:  integer out of range", "(출력은 DB 와 버전에 따라 다름)", 1)

func blockingAt(quote, msg string) ReviewIssue {
	return ReviewIssue{Severity: "major", Gate: GateBlocking, Kind: "factual_error", Section: "walkthrough", Quote: quote,
		Message: msg, Confidence: 0.9, ReaderImpact: "독자가 틀린 결과를 믿는다"}
}

// The previous finding is resolved and the recheck's new finding is in an
// unchanged block: it is ignored and the rewrite is published after one
// targeted recheck (no second full review).
func TestTargetedRecheckIgnoresUnchangedBlocks(t *testing.T) {
	c := &convergeFake{fakeSteps: &fakeSteps{drafts: []string{validMarkup, fixedLog}, reviews: []ReviewOutcome{blockingFact}},
		rechecks: []RecheckOutcome{{Summary: "해결", Previous: []RecheckVerdict{{ID: 1, Resolved: true, Evidence: "조건을 붙임"}},
			NewIssues: []ReviewIssue{blockingAt(fillerSentence, "바뀌지 않은 문장 트집")}}}}
	res, err := RunReviewLoop(context.Background(), 3, c.funcs())
	if err != nil || res.Status != "published" || res.Final != 1 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if c.rev != 1 || len(c.recheckIn) != 1 {
		t.Fatalf("one full review then one targeted recheck: reviews=%d rechecks=%d", c.rev, len(c.recheckIn))
	}
	in := c.recheckIn[0]
	if len(in.Findings) != 1 || len(in.Changed) != 1 || !strings.Contains(in.Changed[0].Text, "버전에 따라 다름") ||
		!strings.Contains(strings.Join(in.Findings[0].Before, ""), "integer out of range") ||
		!strings.Contains(strings.Join(in.Findings[0].After, ""), "버전에 따라 다름") {
		t.Fatalf("recheck input = %+v", in)
	}
	a := res.Attempts[1]
	if a.ReviewMode != ReviewTargeted || len(a.Rechecked) != 1 || !a.Rechecked[0].Resolved {
		t.Fatalf("attempt = mode %s rechecked %+v", a.ReviewMode, a.Rechecked)
	}
	log := res.AttemptLog()
	if log[1].ReviewMode != ReviewTargeted || !strings.Contains(strings.Join(log[1].Recheck, ""), "[resolved]") ||
		!strings.Contains(strings.Join(log[1].Advisory, ""), DowngradeOutsideChange) {
		t.Fatalf("attempt log = %+v", log[1])
	}
	if log[0].ReviewMode != ReviewFull || !strings.Contains(strings.Join(log[0].Adjudication, ""), "[uphold]") {
		t.Fatalf("first attempt: full review, adjudicated finding logged: %+v", log[0])
	}
	// The upheld finding is not adjudicated again; the outside one is never adjudicated.
	if len(c.adjudicated) != 1 {
		t.Fatalf("adjudications = %v", c.adjudicated)
	}
}

// A new blocking finding inside a changed block stops publishing and is
// rewritten; an unresolved previous finding stays.
func TestTargetedRecheckNewBlockingInChangedBlock(t *testing.T) {
	c := &convergeFake{fakeSteps: &fakeSteps{drafts: []string{validMarkup, fixedLog, validMarkup}, reviews: []ReviewOutcome{blockingFact}},
		rechecks: []RecheckOutcome{
			{Previous: []RecheckVerdict{{ID: 1, Resolved: true}}, NewIssues: []ReviewIssue{blockingAt("(출력은 DB 와 버전에 따라 다름)", "새 출력 예시가 틀림")}},
			{Previous: []RecheckVerdict{{ID: 1, Resolved: true}}},
		}}
	res, err := RunReviewLoop(context.Background(), 3, c.funcs())
	if err != nil || res.Status != "published" || res.Final != 2 || len(res.Attempts) != 3 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if res.AttemptLog()[1].Result != "review_failed" || !strings.Contains(c.notes[2], "새 출력 예시가 틀림") {
		t.Fatalf("the new finding must fail attempt 2 and reach the rewrite: %q", c.notes[2])
	}

	// No verdict on the previous finding = unresolved.
	c = &convergeFake{fakeSteps: &fakeSteps{drafts: []string{validMarkup, fixedLog}, reviews: []ReviewOutcome{blockingFact}},
		rechecks: []RecheckOutcome{{Summary: "판정 누락"}}}
	res, err = RunReviewLoop(context.Background(), 2, c.funcs())
	if err != nil || res.Status != "needs_revision" || !res.Attempts[1].Review.Blocking() {
		t.Fatalf("a finding without verdict stays: res=%+v err=%v", res, err)
	}
}

// A blocking finding without confidence ≥ 0.7 or a reader impact is advisory.
func TestBlockingNeedsConfidenceAndReaderImpact(t *testing.T) {
	low := blockingAt("ERROR:  integer out of range", "렌더러 버전 확인 불가")
	low.Confidence = 0.5
	noImpact := blockingAt("ERROR:  integer out of range", "빌드 옵션에 따라 다를 수 있음")
	noImpact.ReaderImpact = " "
	for _, i := range []ReviewIssue{low, noImpact, {Severity: "major", Gate: GateBlocking, Kind: "factual_error", Message: "m"}} {
		if i.IsBlocking() || !i.downgraded() {
			t.Errorf("must be downgraded: %+v (%s)", i, i.Downgrade())
		}
	}
	r := ParseReview(`{"verdict":"fail","issues":[{"severity":"major","gate":"blocking","kind":"factual_error","message":"m","confidence":"0.9","reader_impact":"틀린 판단"}]}`)
	if !r.Blocking() || float64(r.Issues[0].Confidence) != 0.9 {
		t.Fatalf("a quoted confidence is read: %+v", r)
	}

	c := &convergeFake{fakeSteps: &fakeSteps{drafts: []string{validMarkup}, reviews: []ReviewOutcome{
		{Verdict: "fail", Issues: []ReviewIssue{low, noImpact}}}}}
	res, err := RunReviewLoop(context.Background(), 3, c.funcs())
	if err != nil || res.Status != "published" || len(res.Attempts) != 1 || len(c.adjudicated) != 0 {
		t.Fatalf("advisory only → published at once (instant): res=%+v err=%v", res, err)
	}
	adv := strings.Join(res.AttemptLog()[0].Advisory, " ")
	if !strings.Contains(adv, DowngradeLowConfidence) || !strings.Contains(adv, DowngradeNoReaderImpact) {
		t.Fatalf("downgrades must be logged: %s", adv)
	}
}

// A dismissed finding becomes advisory (published); an upheld one is rewritten.
func TestAdjudicationDismissPublishesUpholdRewrites(t *testing.T) {
	c := &convergeFake{fakeSteps: &fakeSteps{drafts: []string{validMarkup}, reviews: []ReviewOutcome{blockingFact}},
		decisions: map[string]string{"출력이 실제와 다름": AdjudicationDismiss}}
	res, err := RunReviewLoop(context.Background(), 3, c.funcs())
	if err != nil || res.Status != "published" || len(res.Attempts) != 1 {
		t.Fatalf("dismissed → published: res=%+v err=%v", res, err)
	}
	log := res.AttemptLog()[0]
	if !strings.Contains(strings.Join(log.Adjudication, ""), "[dismiss]") || !strings.Contains(strings.Join(log.Advisory, ""), DowngradeDismissed) {
		t.Fatalf("attempt log = %+v", log)
	}

	c = &convergeFake{fakeSteps: &fakeSteps{drafts: []string{validMarkup, fixedLog}, reviews: []ReviewOutcome{blockingFact}},
		rechecks: []RecheckOutcome{{Previous: []RecheckVerdict{{ID: 1, Resolved: true}}}}}
	res, err = RunReviewLoop(context.Background(), 3, c.funcs())
	if err != nil || res.Status != "published" || res.Final != 1 || !strings.Contains(c.notes[1], "출력이 실제와 다름") {
		t.Fatalf("upheld → rewrite with the finding: res=%+v err=%v notes=%q", res, err, c.notes)
	}
	// At most maxAdjudications calls; the rest stand.
	many := ReviewOutcome{Verdict: "fail"}
	for _, m := range []string{"a", "b", "c", "d"} {
		many.Issues = append(many.Issues, blockingAt("", m))
	}
	c = &convergeFake{fakeSteps: &fakeSteps{drafts: []string{validMarkup}, reviews: []ReviewOutcome{many}},
		decisions: map[string]string{"a": AdjudicationDismiss, "b": AdjudicationDismiss, "c": AdjudicationDismiss, "d": AdjudicationDismiss}}
	res, _ = RunReviewLoop(context.Background(), 1, c.funcs())
	if len(c.adjudicated) != maxAdjudications || res.Status != "needs_revision" {
		t.Fatalf("adjudicated %v, status %s", c.adjudicated, res.Status)
	}
}

// A rewrite that rewrote more than half of its blocks gets a full review.
func TestLargeRewriteFallsBackToFullReview(t *testing.T) {
	big := validMarkup
	for _, r := range [][2]string{{fillerSentence, "고정 폭 정수는 담을 수 있는 범위가 정해져 있습니다."},
		{"0110 + 0011 의 4비트 결과는?", "0101 + 0011 의 결과는?"}, {"1001 입니다.", "1000 입니다."},
		{"SELECT 2147483647::int + 1;", "SELECT 2147483647::int + 2;"}, {"위 쿼리는 무엇을 출력할까요?", "결과를 예측해 보세요."},
		{"아래 풀이에서 틀린 곳을 찾으세요.", "틀린 곳은?"}, {"자리올림과 넘침을 혼동했습니다.", "넘침을 놓쳤습니다."},
		{"ERROR:  integer out of range", "(출력은 DB 와 버전에 따라 다름)"}} {
		big = strings.ReplaceAll(big, r[0], r[1])
	}
	c := &convergeFake{fakeSteps: &fakeSteps{drafts: []string{validMarkup, big}, reviews: []ReviewOutcome{blockingFact, cleanPass}}}
	res, err := RunReviewLoop(context.Background(), 3, c.funcs())
	if err != nil || res.Status != "published" || res.Final != 1 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if d := res.Attempts[1].Drift; d == nil || d.RewrittenRatio() <= maxTargetedRatio {
		t.Fatalf("drift = %+v", d)
	}
	if c.rev != 2 || len(c.recheckIn) != 0 || res.Attempts[1].ReviewMode != ReviewFullFallback {
		t.Fatalf("full review again: reviews=%d rechecks=%d mode=%s", c.rev, len(c.recheckIn), res.Attempts[1].ReviewMode)
	}

	// An unreadable recheck falls back to a full review too.
	c = &convergeFake{fakeSteps: &fakeSteps{drafts: []string{validMarkup, fixedLog}, reviews: []ReviewOutcome{blockingFact, cleanPass}},
		rechecks: []RecheckOutcome{ParseRecheck("모르겠음")}}
	res, err = RunReviewLoop(context.Background(), 3, c.funcs())
	if err != nil || res.Status != "published" || c.rev != 2 || res.Attempts[1].ReviewMode != ReviewFullFallback {
		t.Fatalf("res=%+v err=%v reviews=%d", res, err, c.rev)
	}
}

// The last rewrite gets the blocking findings only; earlier ones some advisory too.
func TestRevisionNotesAdvisoryOnlyWithAttemptsLeft(t *testing.T) {
	a := Attempt{Reviewed: true, Review: ReviewOutcome{Verdict: "fail", Issues: []ReviewIssue{
		blockingAt("q", "막을 것"), {Severity: "major", Gate: GateAdvisory, Kind: "pedagogy", Message: "개선할 것"},
		func() ReviewIssue {
			i := blockingAt("q", "내려간 것")
			i.Adjudication = AdjudicationDismiss
			return i
		}(),
	}}}
	with, without := RevisionNotes(a, true), RevisionNotes(a, false)
	if !strings.Contains(with, "개선할 것") || strings.Contains(without, "개선할 것") || !strings.Contains(without, "막을 것") {
		t.Fatalf("with=%q without=%q", with, without)
	}
	if strings.Contains(with, "내려간 것") {
		t.Fatal("a dismissed finding is not sent back to the rewrite")
	}
}

// A block longer than the quote bound is cut with an explicit marker, never
// silently, so the reviewer cannot mistake the cut for a truncated sentence.
func TestQuoteBlockMarksCut(t *testing.T) {
	short := lecture.Block{Type: lecture.BlockMarkdown, Text: "짧은 문단입니다."}
	if got := quoteBlock(short); strings.Contains(got, "이하 생략") {
		t.Fatalf("short block marked as cut: %q", got)
	}
	long := lecture.Block{Type: lecture.BlockMarkdown, Text: strings.Repeat("가", maxQuotedBlockRune+500)}
	got := quoteBlock(long)
	if !strings.HasSuffix(got, quotedCutMarker) {
		t.Fatalf("long block not marked: %q", got[len(got)-80:])
	}
}
