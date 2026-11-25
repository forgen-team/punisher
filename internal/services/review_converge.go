package services

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/forgen-team/punisher/internal/lecture"
	"github.com/forgen-team/punisher/internal/llmjson"
	"github.com/forgen-team/punisher/internal/prompts"
)

// How a draft was reviewed (Attempt.ReviewMode).
const (
	ReviewFull         = "full"
	ReviewTargeted     = "targeted"
	ReviewFullFallback = "full_fallback"
)

// maxTargetedRatio: a partial rewrite that rewrote more than this share of
// its blocks gets a full review again instead of a targeted recheck.
const maxTargetedRatio = 0.5

// maxAdjudications bounds the adjudication calls per review; further blocking
// findings stand as reviewed.
const maxAdjudications = 3

// Prompt size bounds of the blocks quoted to a recheck or an adjudication.
const (
	maxQuotedBlocks    = 4
	maxQuotedBlockRune = 8000
)

// quotedCutMarker marks a block shortened for the prompt; unmarked cuts were
// judged as truncated lecture sentences and blocked publishing.
const quotedCutMarker = "\n…(검수 입력 길이 제한으로 이하 생략 — 강의 원문은 잘리지 않았습니다)"

// quoteBlock renders one block for a recheck/adjudication prompt, marking any cut.
func quoteBlock(b lecture.Block) string {
	text := lecture.RenderBlock(b)
	cut := truncateRunes(text, maxQuotedBlockRune)
	if cut == text {
		return text
	}
	return cut + quotedCutMarker
}

// RecheckInput is what a targeted recheck reads: the previous blocking
// findings with their blocks before and after the rewrite, and the blocks the
// rewrite changed.
type RecheckInput struct {
	Findings []prompts.RecheckFinding
	Changed  []prompts.RecheckBlock
}

// RecheckVerdict is the recheck's verdict on one previous blocking finding
// (ID is 1-based in RecheckInput.Findings).
type RecheckVerdict struct {
	ID       int         `json:"id"`
	Resolved bool        `json:"resolved"`
	Evidence string      `json:"evidence"`
	Finding  ReviewIssue `json:"-"`
}

// RecheckOutcome is the parsed targeted recheck. Unparsed: the output could
// not be read (the loop falls back to a full review).
type RecheckOutcome struct {
	Summary   string           `json:"summary"`
	Previous  []RecheckVerdict `json:"previous"`
	NewIssues []ReviewIssue    `json:"new_issues"`
	Unparsed  bool             `json:"-"`
}

// ParseRecheck parses recheck output.
func ParseRecheck(text string) RecheckOutcome {
	out, err := llmjson.TryParseJSON[RecheckOutcome](text)
	if err != nil {
		return RecheckOutcome{Unparsed: true}
	}
	return out
}

// AdjudicationInput is one blocking finding and the blocks holding its sentence.
type AdjudicationInput struct {
	Issue  ReviewIssue
	Blocks []string
}

// AdjudicationResult is the second opinion on one blocking finding.
type AdjudicationResult struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// ParseAdjudication parses adjudication output; unreadable output has no
// decision (the finding is upheld).
func ParseAdjudication(text string) AdjudicationResult {
	out, err := llmjson.TryParseJSON[AdjudicationResult](text)
	if err != nil {
		return AdjudicationResult{}
	}
	out.Decision = strings.ToLower(strings.TrimSpace(out.Decision))
	return out
}

// judged: the review produced a verdict worth keeping (not an unreadable or
// finding-less fail), so a rewrite of the draft may be rechecked against it.
func (r ReviewOutcome) judged() bool { return r.Verdict == "pass" || len(r.Issues) > 0 }

// reviewAttempt reviews a parsed draft. A partial rewrite of a reviewed draft
// (base) gets a targeted recheck unless it rewrote more than maxTargetedRatio
// of its blocks or the recheck output is unreadable (full review again); any
// other draft gets a full review. Blocking findings are then adjudicated.
func (f LoopFuncs) reviewAttempt(ctx context.Context, a *Attempt, base *Attempt) error {
	mode := ReviewFull
	if base != nil && base.Reviewed && base.Review.judged() && f.Recheck != nil && a.Drift != nil {
		if a.Drift.RewrittenRatio() <= maxTargetedRatio {
			ok, err := f.targetedReview(ctx, a, *base)
			if err != nil {
				return err
			}
			if ok {
				a.ReviewMode = ReviewTargeted
				f.adjudicate(ctx, a)
				return nil
			}
		}
		mode = ReviewFullFallback
	}
	r, err := f.review(ctx, a.Markup)
	if err != nil {
		return err
	}
	a.Review, a.ReviewMode = r, mode
	f.adjudicate(ctx, a)
	return nil
}

// targetedReview rechecks a partial rewrite against base's review: each
// previous blocking finding is resolved or not, new blocking findings count
// only inside the rewritten blocks, and base's other findings are kept where
// their sentence is still there. false = the recheck output was unreadable.
func (f LoopFuncs) targetedReview(ctx context.Context, a *Attempt, base Attempt) (bool, error) {
	rewritten := rewrittenBlocks(base.Content, a.Content)
	var prev []ReviewIssue
	for _, i := range base.Review.Issues {
		if i.IsBlocking() {
			prev = append(prev, i)
		}
	}
	var out RecheckOutcome
	if len(rewritten) > 0 {
		in := RecheckInput{}
		for k, i := range prev {
			in.Findings = append(in.Findings, prompts.RecheckFinding{ID: k + 1, Kind: i.Kind, Section: i.Section,
				Quote: i.Quote, Message: i.Message,
				Before: renderBlocks(base.Content, quoteBlocks(base.Content, i.Quote)),
				After:  afterBlocks(base.Content, a.Content, i, rewritten)})
		}
		for _, p := range orderedPositions(a.Content, rewritten) {
			in.Changed = append(in.Changed, prompts.RecheckBlock{Section: a.Content.Sections[p[0]].Kind,
				Text: renderBlocks(a.Content, [][2]int{p})[0]})
		}
		cctx, cancel := withLimit(ctx, f.ReviewTimeout)
		var err error
		out, err = f.Recheck(cctx, in)
		cancel()
		if err != nil {
			return false, err
		}
		if out.Unparsed {
			logf("lecture recheck of attempt %d unreadable: full review", a.Number)
			return false, nil
		}
	}
	a.Review, a.Rechecked = mergeRecheck(base, a.Content, prev, rewritten, out)
	return true, nil
}

// mergeRecheck builds the rewrite's review from base's and the recheck:
// unresolved previous blocking findings stay, new findings outside the
// rewritten blocks are downgraded (OutsideChange), and base's non-blocking
// findings and reader gaps are kept while their sentence is still there.
func mergeRecheck(base Attempt, next lecture.Content, prev []ReviewIssue, rewritten map[[2]int]bool, out RecheckOutcome) (ReviewOutcome, []RecheckVerdict) {
	byID := map[int]RecheckVerdict{}
	for _, v := range out.Previous {
		byID[v.ID] = v
	}
	r := ReviewOutcome{Summary: out.Summary, Issues: []ReviewIssue{}, ReaderGaps: []ReaderGap{},
		Independent: base.Review.Independent, Links: base.Review.Links}
	for _, sc := range base.Review.Rubric { // 바뀌지 않은 문장에 근거한 점수는 처음 감수 판정을 그대로 쓴다
		if stillThere(next, sc.Quote) {
			r.Rubric = append(r.Rubric, sc)
		}
	}
	var verdicts []RecheckVerdict
	for k, i := range prev {
		v, ok := byID[k+1]
		switch {
		case len(rewritten) == 0:
			v = RecheckVerdict{Evidence: "재작성에서 바뀐 블록이 없음"}
		case !ok:
			v = RecheckVerdict{Evidence: "재검수 결과에 이 지적의 판정이 없어 미해결로 봄"}
		}
		v.ID, v.Finding = k+1, i
		verdicts = append(verdicts, v)
		if !v.Resolved {
			r.Issues = append(r.Issues, i)
		}
	}
	for _, i := range out.NewIssues {
		if !inRewritten(next, rewritten, i) {
			i.OutsideChange = true
		}
		r.Issues = append(r.Issues, i)
	}
	for _, i := range base.Review.Issues {
		if !i.IsBlocking() && stillThere(next, i.Quote) {
			r.Issues = append(r.Issues, i)
		}
	}
	for _, g := range base.Review.ReaderGaps {
		if stillThere(next, g.Quote) {
			r.ReaderGaps = append(r.ReaderGaps, g)
		}
	}
	r.Verdict = "pass"
	for _, i := range r.Issues {
		if i.IsBlocking() {
			r.Verdict = "fail"
		}
	}
	if r.Summary == "" {
		r.Summary = "표적 재검수: 바뀐 블록과 이전 차단 지적만 다시 봄"
	}
	return r, verdicts
}

// inRewritten: the finding's sentence is in a rewritten block. A quote not
// found verbatim falls back to its section (any rewritten block there); a
// finding outside the four sections (header) is kept.
func inRewritten(next lecture.Content, rewritten map[[2]int]bool, i ReviewIssue) bool {
	if ps := quoteBlocks(next, i.Quote); len(ps) > 0 {
		for _, p := range ps {
			if rewritten[p] {
				return true
			}
		}
		return false
	}
	si := sectionIndex(next, i.Section)
	if si < 0 {
		return true
	}
	for p := range rewritten {
		if p[0] == si {
			return true
		}
	}
	return false
}

// stillThere: an empty quote, or a quote still found in the lecture.
func stillThere(c lecture.Content, quote string) bool {
	return strings.TrimSpace(quote) == "" || len(quoteBlocks(c, quote)) > 0
}

// afterBlocks are the blocks of next that stand for what finding i pointed at
// in prev: the same block when it was kept, else the rewritten block at the
// same position or the section's rewritten blocks.
func afterBlocks(prev, next lecture.Content, i ReviewIssue, rewritten map[[2]int]bool) []string {
	kept := sectionBlockKeys(next)
	seen := map[[2]int]bool{}
	var out []string
	add := func(p [2]int) {
		if !seen[p] {
			seen[p] = true
			out = append(out, renderBlocks(next, [][2]int{p})...)
		}
	}
	sectionRewrites := func(si int) {
		for _, p := range orderedPositions(next, rewritten) {
			if p[0] == si {
				add(p)
			}
		}
	}
	before := quoteBlocks(prev, i.Quote)
	for _, p := range before {
		s := prev.Sections[p[0]]
		blk := s.Blocks[p[1]]
		if kept[s.Kind+"\x00"+blockKey(blk)] {
			out = append(out, "(바뀌지 않음)\n"+lecture.RenderBlock(blk))
			continue
		}
		si := sectionIndex(next, s.Kind)
		if si >= 0 && rewritten[[2]int{si, p[1]}] {
			add([2]int{si, p[1]})
			continue
		}
		sectionRewrites(si)
	}
	if len(before) == 0 {
		sectionRewrites(sectionIndex(next, i.Section))
	}
	if len(out) > maxQuotedBlocks {
		out = out[:maxQuotedBlocks]
	}
	return out
}

// adjudicate asks for a second opinion on the review's blocking findings not
// adjudicated yet (most severe first, at most maxAdjudications, in parallel):
// dismissed ones become advisory. A failed or unreadable adjudication upholds
// the finding.
func (f LoopFuncs) adjudicate(ctx context.Context, a *Attempt) {
	if f.Adjudicate == nil {
		return
	}
	var pending []int
	for k, i := range a.Review.Issues {
		if i.IsBlocking() && i.Adjudication == "" {
			pending = append(pending, k)
		}
	}
	sort.SliceStable(pending, func(x, y int) bool {
		return severityRank(a.Review.Issues[pending[x]].Severity) < severityRank(a.Review.Issues[pending[y]].Severity)
	})
	if len(pending) > maxAdjudications {
		pending = pending[:maxAdjudications]
	}
	if len(pending) == 0 {
		return
	}
	// The verdicts are written on a.Review's own copy (the slice may be shared
	// with the review that produced it).
	a.Review.Issues = append([]ReviewIssue(nil), a.Review.Issues...)
	results := make([]AdjudicationResult, len(pending))
	errs := make([]error, len(pending))
	var wg sync.WaitGroup
	for k, idx := range pending {
		wg.Add(1)
		go func(k int, i ReviewIssue) {
			defer wg.Done()
			ps := quoteBlocks(a.Content, i.Quote)
			if len(ps) == 0 {
				ps = sectionPositions(a.Content, i.Section)
			}
			cctx, cancel := withLimit(ctx, f.AdjudicateTimeout)
			defer cancel()
			results[k], errs[k] = f.Adjudicate(cctx, AdjudicationInput{Issue: i, Blocks: renderBlocks(a.Content, ps)})
		}(k, a.Review.Issues[idx])
	}
	wg.Wait()
	for k, idx := range pending {
		i := &a.Review.Issues[idx]
		switch {
		case errs[k] != nil:
			i.Adjudication, i.AdjudicationReason = AdjudicationUphold, "재판정 실패로 지적 유지: "+errs[k].Error()
		case results[k].Decision == AdjudicationDismiss:
			i.Adjudication, i.AdjudicationReason = AdjudicationDismiss, results[k].Reason
		case results[k].Decision == AdjudicationUphold:
			i.Adjudication, i.AdjudicationReason = AdjudicationUphold, results[k].Reason
		default:
			i.Adjudication, i.AdjudicationReason = AdjudicationUphold, "재판정 결과를 해석할 수 없어 지적 유지"
		}
	}
	if !a.Review.Blocking() {
		a.Review.Verdict = "pass"
	}
}

func severityRank(s string) int {
	switch s {
	case "critical":
		return 0
	case "major":
		return 1
	case "minor":
		return 2
	}
	return 3
}

// sectionIndex is the index of the section of that kind (-1 when none).
func sectionIndex(c lecture.Content, kind string) int {
	for i, s := range c.Sections {
		if s.Kind == kind {
			return i
		}
	}
	return -1
}

// sectionPositions are the block positions of the section of that kind.
func sectionPositions(c lecture.Content, kind string) [][2]int {
	si := sectionIndex(c, kind)
	if si < 0 {
		return nil
	}
	out := make([][2]int, 0, len(c.Sections[si].Blocks))
	for j := range c.Sections[si].Blocks {
		out = append(out, [2]int{si, j})
	}
	return out
}

// orderedPositions lists the positions of set in document order.
func orderedPositions(c lecture.Content, set map[[2]int]bool) [][2]int {
	var out [][2]int
	for i, s := range c.Sections {
		for j := range s.Blocks {
			if set[[2]int{i, j}] {
				out = append(out, [2]int{i, j})
			}
		}
	}
	return out
}

// renderBlocks renders the blocks at ps as markup (at most maxQuotedBlocks,
// each cut at maxQuotedBlockRune).
func renderBlocks(c lecture.Content, ps [][2]int) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		if len(out) == maxQuotedBlocks {
			break
		}
		out = append(out, quoteBlock(c.Sections[p[0]].Blocks[p[1]]))
	}
	return out
}
