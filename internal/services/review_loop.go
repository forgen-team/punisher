package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/store"
	"github.com/wooo-jin/punisher/internal/llm"
	"github.com/wooo-jin/punisher/internal/llmjson"
)

// Review gates: a blocking finding stops publishing (factual error,
// misleading explanation, unresolved prescription, error/output mismatch); an
// advisory one is a teaching-quality improvement that never blocks.
const (
	GateBlocking = "blocking"
	GateAdvisory = "advisory"
)

// advisoryKinds never block publishing, whatever gate the reviewer wrote:
// composition, learning activities, term definitions, coverage, sources.
var advisoryKinds = map[string]bool{
	"structure": true, "pedagogy": true, "omission": true, "assumed_knowledge": true,
	"terminology": true, "source": true,
}

// ReviewIssue is one reviewer finding.
type ReviewIssue struct {
	Severity   string `json:"severity"`
	Gate       string `json:"gate"`
	Kind       string `json:"kind"`
	Section    string `json:"section"`
	Quote      string `json:"quote"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

// IsBlocking: the reviewer's gate decides for the factual kinds
// (factual_error, overclaim, typo — and an unknown kind, to stay safe); a
// review without gates (an older user-edited review prompt) falls back to
// critical/major. Advisory kinds never block.
func (i ReviewIssue) IsBlocking() bool {
	if advisoryKinds[i.Kind] {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(i.Gate)) {
	case GateBlocking:
		return true
	case GateAdvisory:
		return false
	}
	return i.Severity == "critical" || i.Severity == "major"
}

// ReaderGap is a sentence a web FE/BE developer could not follow because it
// assumes an undefined term or mechanism (reviewer's beginner-reader pass).
// Reader gaps are always advisory.
type ReaderGap struct {
	Section string `json:"section"`
	Quote   string `json:"quote"`
	Unknown string `json:"unknown"`
	Why     string `json:"why"`
	Fix     string `json:"fix"`
}

// ReviewOutcome is the parsed review pass.
type ReviewOutcome struct {
	Verdict    string        `json:"verdict"`
	Summary    string        `json:"summary"`
	ReaderGaps []ReaderGap   `json:"reader_gaps"`
	Issues     []ReviewIssue `json:"issues"`
}

// Blocking reports whether the review stops publishing: any blocking finding
// (the verdict is not trusted on its own), or a fail with no finding at all
// (including unparseable output) since nothing tells what is wrong. A fail
// with only advisory findings does not block.
func (r ReviewOutcome) Blocking() bool {
	for _, i := range r.Issues {
		if i.IsBlocking() {
			return true
		}
	}
	return r.Verdict != "pass" && len(r.Issues) == 0
}

// advisoryToRevise: advisory findings worth one rewrite (critical/major;
// minor findings and reader gaps alone never trigger a rewrite).
func (r ReviewOutcome) advisoryToRevise() bool {
	for _, i := range r.Issues {
		if !i.IsBlocking() && (i.Severity == "critical" || i.Severity == "major") {
			return true
		}
	}
	return false
}

// ParseReview parses reviewer output. Unparseable output is a failing review
// (never silently a pass).
func ParseReview(text string) ReviewOutcome {
	out, err := llmjson.TryParseJSON[ReviewOutcome](text)
	if err != nil || out.Verdict == "" {
		return ReviewOutcome{Verdict: "fail", Summary: "검수 결과를 해석할 수 없어 통과 처리하지 않았습니다."}
	}
	if out.Issues == nil {
		out.Issues = []ReviewIssue{}
	}
	if out.ReaderGaps == nil {
		out.ReaderGaps = []ReaderGap{}
	}
	return out
}

// Attempt is one generate → validate → review round.
type Attempt struct {
	Number   int
	Markup   string
	Content  lecture.Content
	ParseErr error
	Issues   []lecture.Issue
	Review   ReviewOutcome
	Reviewed bool
	// Err is the LLM error that ended this attempt (an attempt time limit or
	// a provider error); the attempt is failed, the loop may go on.
	Err      error
	Duration time.Duration
	// Drift compares a partial rewrite with the draft it revised (nil for a
	// first draft or a full rewrite).
	Drift *Drift
}

// drafted: the attempt produced markup that parsed.
func (a Attempt) drafted() bool { return a.Markup != "" && a.ParseErr == nil }

// blockingIssues are the format / data / leak issues that block publishing.
func (a Attempt) blockingIssues() []lecture.Issue { return lecture.BlockingIssues(a.Issues) }

// Passed: publishable — parsed, no blocking validation issue, reviewed and
// no blocking review finding. Advisory findings may remain.
func (a Attempt) Passed() bool {
	return a.Err == nil && a.drafted() && len(a.blockingIssues()) == 0 && a.Reviewed && !a.Review.Blocking()
}

// hasAdvisory: a passed attempt still has advisory findings worth a rewrite.
func (a Attempt) hasAdvisory() bool {
	return len(a.Issues) > len(a.blockingIssues()) || a.Review.advisoryToRevise()
}

// Record summarizes the attempt for generation_jobs.attempt_log.
func (a Attempt) Record() store.AttemptRecord {
	r := store.AttemptRecord{Attempt: a.Number, Result: "ok", DurationMs: a.Duration.Milliseconds()}
	switch {
	case a.Err != nil && isAttemptTimeout(a.Err):
		r.Result, r.Message = "timeout", a.Err.Error()
	case a.Err != nil:
		r.Result, r.Message = "llm_error", a.Err.Error()
	case a.ParseErr != nil:
		r.Result, r.Message = "parse_error", a.ParseErr.Error()
	case len(a.blockingIssues()) > 0:
		msgs := make([]string, 0, len(a.Issues))
		for _, i := range a.blockingIssues() {
			msgs = append(msgs, i.String())
		}
		r.Result, r.Message = "invalid", strings.Join(msgs, "; ")
	case a.Reviewed && a.Review.Blocking():
		msgs := []string{a.Review.Summary}
		for _, i := range a.Review.Issues {
			if i.IsBlocking() {
				msgs = append(msgs, fmt.Sprintf("[%s] %s", i.Kind, i.Message))
			}
		}
		r.Result, r.Message = "review_failed", strings.Join(msgs, " | ")
	case a.Reviewed:
		r.Message = a.Review.Summary
	}
	r.Message = truncateRunes(r.Message, 1000)
	r.Advisory = a.advisoryRecords()
	if a.Drift != nil {
		r.Drift = a.Drift.String()
	}
	return r
}

// maxAdvisoryRecords bounds the advisory lines kept per attempt.
const maxAdvisoryRecords = 12

// advisoryRecords lists the attempt's advisory findings (machine checks and
// review) for the attempt log.
func (a Attempt) advisoryRecords() []string {
	var out []string
	for _, i := range a.Issues {
		if i.Advisory {
			out = append(out, truncateRunes("[machine] "+i.String(), 300))
		}
	}
	for _, i := range a.Review.Issues {
		if !i.IsBlocking() {
			out = append(out, truncateRunes(fmt.Sprintf("[%s/%s] %s", i.Kind, i.Severity, i.Message), 300))
		}
	}
	if len(out) > maxAdvisoryRecords {
		out = append(out[:maxAdvisoryRecords], fmt.Sprintf("… 외 %d건", len(out)-maxAdvisoryRecords))
	}
	return out
}

// AttemptLog is the per-attempt record of a loop result; the published
// attempt is marked.
func (r LoopResult) AttemptLog() []store.AttemptRecord {
	out := make([]store.AttemptRecord, 0, len(r.Attempts))
	for i, a := range r.Attempts {
		rec := a.Record()
		rec.Published = r.Status == "published" && i == r.Final
		out = append(out, rec)
	}
	return out
}

// LoopResult is the outcome of the review loop.
type LoopResult struct {
	Attempts []Attempt
	// Final indexes the attempt to store as the live version: the published
	// attempt, else the last parsed one; -1 if none parsed.
	Final  int
	Status string // published | needs_revision
}

// Revision is what a rewrite gets: the findings to fix and, for a partial
// rewrite, the previous draft to edit in place ("" = write from scratch).
type Revision struct {
	Notes    string
	Previous string
}

// LoopFuncs are the steps of the loop (LLM-backed in production, fakes in tests).
type LoopFuncs struct {
	Generate func(ctx context.Context, rev Revision) (string, error)
	Review   func(ctx context.Context, markup string) (ReviewOutcome, error)
	// Parse: format validation and machine checks (issues = rewrite findings;
	// only non-advisory ones block).
	Parse func(markup string) (lecture.Content, []lecture.Issue, error)
	// OnStage (optional) reports each step: writing/rewriting (attempt = rewrite
	// number), checking, reviewing — the instant stream's stage events.
	OnStage func(stage string, attempt int)
	// SkipReviewOnIssues: a draft with blocking format/machine issues is
	// rewritten without a review call (the instant pipeline: format → machine → review).
	SkipReviewOnIssues bool
	// DraftTimeout / ReviewTimeout bound one attempt's draft and review calls
	// (0 = no limit beyond ctx). A call over its limit fails that attempt only.
	DraftTimeout, ReviewTimeout time.Duration
}

func (f LoopFuncs) stage(stage string, attempt int) {
	if f.OnStage != nil {
		f.OnStage(stage, attempt)
	}
}

func withLimit(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, d)
}

func (f LoopFuncs) generate(ctx context.Context, rev Revision) (string, error) {
	cctx, cancel := withLimit(ctx, f.DraftTimeout)
	defer cancel()
	return f.Generate(cctx, rev)
}

func (f LoopFuncs) review(ctx context.Context, markup string) (ReviewOutcome, error) {
	cctx, cancel := withLimit(ctx, f.ReviewTimeout)
	defer cancel()
	return f.Review(cctx, markup)
}

// isAttemptTimeout: the error is a per-call time limit (ours or the
// provider's, including a queue wait), not a cancelled job.
func isAttemptTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, llm.ErrTimeout)
}

// RunReviewLoop generates a draft, validates and reviews it, and rewrites it
// with the findings, up to maxAttempts drafts in all (library 2 = one
// regeneration, instant 3 = two rewrites).
//
//   - Only blocking findings (factual / misleading / error-output mismatch,
//     format, block data, leaks) fail an attempt. A draft with advisory
//     findings only is publishable; while attempts remain it gets one
//     advisory rewrite, and if that rewrite is not publishable (a new blocking
//     finding, a timeout, an error) the earlier passing draft is published.
//   - Rewrites are partial: the previous parsed draft is given to edit in
//     place, and the change outside the findings is measured (Attempt.Drift).
//   - A call over its attempt limit fails that attempt only; the loop goes on
//     (or publishes the passing draft it has). Any other LLM error aborts with
//     that error unless a passing draft exists.
//
// If no attempt passed, Status is needs_revision with Final = the last parsed
// attempt; if none parsed, the last LLM error or ErrUnusableOutput is returned.
func RunReviewLoop(ctx context.Context, maxAttempts int, f LoopFuncs) (LoopResult, error) {
	res := LoopResult{Final: -1}
	rev := Revision{}
	revFrom := -1 // attempt whose draft rev.Previous holds
	best := -1    // latest passing attempt
	var lastErr error
	for n := 1; n <= maxAttempts && ctx.Err() == nil; n++ {
		if n == 1 {
			f.stage(StageWriting, 0)
		} else {
			f.stage(StageRewriting, n-1)
		}
		start := time.Now()
		a := Attempt{Number: n}
		a.Markup, a.Err = f.generate(ctx, rev)
		if a.Err == nil {
			f.stage(StageChecking, n-1)
			a.Content, a.Issues, a.ParseErr = f.Parse(a.Markup)
			if a.ParseErr == nil && revFrom >= 0 {
				a.Drift = measureDrift(res.Attempts[revFrom], a.Content)
				if a.Drift.Warn() {
					logf("lecture rewrite attempt %d changed unrelated blocks: %s", n, a.Drift)
				}
			}
			if a.ParseErr == nil && !(f.SkipReviewOnIssues && len(a.blockingIssues()) > 0) {
				f.stage(StageReviewing, n-1)
				var rerr error
				if a.Review, rerr = f.review(ctx, a.Markup); rerr != nil {
					a.Err = fmt.Errorf("review attempt %d: %w", n, rerr)
				} else {
					a.Reviewed = true
				}
			}
		} else {
			a.Markup = ""
			a.Err = fmt.Errorf("generate attempt %d: %w", n, a.Err)
		}
		a.Duration = time.Since(start)
		res.Attempts = append(res.Attempts, a)
		idx := len(res.Attempts) - 1

		if a.Err != nil {
			lastErr = a.Err
			if best >= 0 {
				break // the passing draft stands
			}
			if ctx.Err() != nil || !isAttemptTimeout(a.Err) {
				return res, a.Err
			}
			continue // this attempt timed out: retry with the same revision
		}
		if a.Passed() {
			if best >= 0 || !a.hasAdvisory() || n == maxAttempts {
				best = idx
				break
			}
			best = idx
		} else if best >= 0 {
			break // the advisory rewrite broke something: keep the passing draft
		}
		rev = Revision{Notes: RevisionNotes(a)}
		if a.drafted() {
			rev.Previous, revFrom = a.Markup, idx
		}
	}
	if best >= 0 {
		res.Final, res.Status = best, "published"
		return res, nil
	}
	for i, a := range res.Attempts {
		if a.drafted() {
			res.Final = i
		}
	}
	if res.Final < 0 {
		if lastErr != nil {
			return res, lastErr
		}
		if err := ctx.Err(); err != nil {
			return res, err
		}
		return res, ErrUnusableOutput
	}
	res.Status = "needs_revision"
	return res, nil
}

// maxGapNotes bounds reader-gap lines in revision notes (prompt size).
const maxGapNotes = 15

// RevisionNotes formats an attempt's problems for the rewrite prompt:
// blocking findings first (must fix), then advisory ones (improve where it
// does not create a new factual problem).
func RevisionNotes(a Attempt) string {
	var must, improve strings.Builder
	switch {
	case errors.Is(a.ParseErr, lecture.ErrIncomplete):
		must.WriteString("- [형식] 출력이 @@end 전에 끊겼습니다(출력 한도). 분량 규칙(본문 약 8,500자)을 지켜 줄이고 마지막 줄에 @@end 를 쓰세요.\n")
	case a.ParseErr != nil:
		fmt.Fprintf(&must, "- [형식] 마크업을 해석할 수 없습니다: %v. 형식 규칙을 정확히 지키고 마지막 줄에 @@end 를 쓰세요.\n", a.ParseErr)
	}
	for _, i := range a.Issues {
		if i.Advisory {
			fmt.Fprintf(&improve, "- [구성] %s\n", i.String())
		} else {
			fmt.Fprintf(&must, "- [스키마] %s\n", i.String())
		}
	}
	for _, i := range a.Review.Issues {
		switch {
		case i.IsBlocking():
			writeReviewNote(&must, i)
		case i.Severity != "minor":
			writeReviewNote(&improve, i)
		}
	}
	for i, g := range a.Review.ReaderGaps {
		if i == maxGapNotes {
			fmt.Fprintf(&improve, "- [전제 지식] 외 %d건도 같은 방식으로 첫 등장에서 정의하세요.\n", len(a.Review.ReaderGaps)-maxGapNotes)
			break
		}
		fmt.Fprintf(&improve, "- [전제 지식 · %s] %q 에서 '%s'를 정의 없이 씀 → %s\n", g.Section, g.Quote, g.Unknown, g.Fix)
	}
	if must.Len() == 0 && a.Reviewed && a.Review.Blocking() {
		fmt.Fprintf(&must, "- [검수] %s\n", a.Review.Summary)
	}
	var b strings.Builder
	if must.Len() > 0 {
		b.WriteString("### 반드시 고칠 것(사실 오류·오도·형식)\n" + must.String())
	}
	if improve.Len() > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("### 개선 권장(고쳐도 새 사실 오류가 생기지 않는 범위에서만)\n" + improve.String())
	}
	return b.String()
}

func writeReviewNote(b *strings.Builder, i ReviewIssue) {
	fmt.Fprintf(b, "- [%s/%s · %s] %s", i.Severity, i.Kind, i.Section, i.Message)
	if i.Quote != "" {
		fmt.Fprintf(b, " (문제 문장: %q)", i.Quote)
	}
	if i.Suggestion != "" {
		fmt.Fprintf(b, " → %s", i.Suggestion)
	}
	b.WriteString("\n")
}

func issuesJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil || string(raw) == "null" {
		return json.RawMessage("[]")
	}
	return raw
}
