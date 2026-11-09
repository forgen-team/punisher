package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/llmjson"
)

// ReviewIssue is one reviewer finding.
type ReviewIssue struct {
	Severity   string `json:"severity"`
	Kind       string `json:"kind"`
	Section    string `json:"section"`
	Quote      string `json:"quote"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

// ReviewOutcome is the parsed review pass.
type ReviewOutcome struct {
	Verdict string        `json:"verdict"`
	Summary string        `json:"summary"`
	Issues  []ReviewIssue `json:"issues"`
}

// Blocking reports whether the review requires revision: explicit fail or
// any critical/major issue (the verdict is not trusted on its own).
func (r ReviewOutcome) Blocking() bool {
	if r.Verdict != "pass" {
		return true
	}
	for _, i := range r.Issues {
		if i.Severity == "critical" || i.Severity == "major" {
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
	return out
}

// Attempt is one generate → review → validate round.
type Attempt struct {
	Number   int
	Markup   string
	Content  lecture.Content
	ParseErr error
	Issues   []lecture.Issue
	Review   ReviewOutcome
	Reviewed bool
}

// Passed: parsed, structurally valid and review not blocking.
func (a Attempt) Passed() bool {
	return a.ParseErr == nil && len(a.Issues) == 0 && a.Reviewed && !a.Review.Blocking()
}

// LoopResult is the outcome of the review loop.
type LoopResult struct {
	Attempts []Attempt
	// Final indexes the attempt to store as the live version; -1 if none parsed.
	Final  int
	Status string // published | needs_revision
}

// LoopFuncs are the steps of the loop (LLM-backed in production, fakes in tests).
type LoopFuncs struct {
	Generate func(ctx context.Context, revisionNotes string) (string, error)
	Review   func(ctx context.Context, markup string) (ReviewOutcome, error)
	Parse    func(markup string) (lecture.Content, []lecture.Issue, error)
}

// RunReviewLoop generates a draft, reviews and validates it; on failure it
// regenerates once with the findings (maxAttempts=2 → one regeneration).
// Any LLM error aborts with that error so the job is recorded as failed and
// nothing is saved. If no attempt parsed, ErrUnusableOutput is returned.
func RunReviewLoop(ctx context.Context, maxAttempts int, f LoopFuncs) (LoopResult, error) {
	res := LoopResult{Final: -1}
	notes := ""
	for n := 1; n <= maxAttempts; n++ {
		markup, err := f.Generate(ctx, notes)
		if err != nil {
			return res, fmt.Errorf("generate attempt %d: %w", n, err)
		}
		a := Attempt{Number: n, Markup: markup}
		a.Content, a.Issues, a.ParseErr = f.Parse(markup)
		if a.ParseErr == nil {
			a.Review, err = f.Review(ctx, markup)
			if err != nil {
				return res, fmt.Errorf("review attempt %d: %w", n, err)
			}
			a.Reviewed = true
			res.Final = len(res.Attempts)
		}
		res.Attempts = append(res.Attempts, a)
		if a.Passed() {
			res.Status = "published"
			return res, nil
		}
		notes = RevisionNotes(a)
	}
	if res.Final < 0 {
		return res, ErrUnusableOutput
	}
	res.Status = "needs_revision"
	return res, nil
}

// RevisionNotes formats an attempt's problems for the regeneration prompt.
func RevisionNotes(a Attempt) string {
	var b strings.Builder
	if a.ParseErr != nil {
		fmt.Fprintf(&b, "- [형식] 마크업을 해석할 수 없습니다: %v. 형식 규칙을 정확히 지키고 마지막 줄에 @@end 를 쓰세요.\n", a.ParseErr)
	}
	for _, i := range a.Issues {
		fmt.Fprintf(&b, "- [스키마] %s\n", i.String())
	}
	for _, i := range a.Review.Issues {
		if i.Severity == "minor" {
			continue
		}
		fmt.Fprintf(&b, "- [%s/%s · %s] %s", i.Severity, i.Kind, i.Section, i.Message)
		if i.Quote != "" {
			fmt.Fprintf(&b, " (문제 문장: %q)", i.Quote)
		}
		if i.Suggestion != "" {
			fmt.Fprintf(&b, " → %s", i.Suggestion)
		}
		b.WriteString("\n")
	}
	if b.Len() == 0 && a.Reviewed && a.Review.Verdict != "pass" {
		fmt.Fprintf(&b, "- [검수] %s\n", a.Review.Summary)
	}
	return b.String()
}

func issuesJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil || string(raw) == "null" {
		return json.RawMessage("[]")
	}
	return raw
}
