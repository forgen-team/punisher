package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/forgen-team/punisher/internal/lecture"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/llmjson"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/quality"
)

// Lecture quality v2 (research 2026-11-25): the deterministic rubric feeds
// rewrites, the review gets what the app computed (SQL runs, arithmetic,
// independent solutions, broken links) as evidence, and every review row
// stores the rubric and the reviewer's 1~4 scores for trends (never a gate).

// verifyTimeout bounds the independent solving pass and the link checks of one review.
const verifyTimeout = 75 * time.Second

// triggerRules are the rubric rules that alone are worth a rewrite (reading
// time over 25 minutes, an SQL example or a calculation that does not hold):
// they are advisory issues of the attempt. noteRules only go into the notes
// of a rewrite that happens anyway (a library lecture is not drafted again
// for spacing or recall prompts alone); the rest (wording, link repeats,
// paragraph length, misconception form, block length) are stored only.
var (
	triggerRules = map[string]bool{"G07": true, "G21": true, "G22": true}
	noteRules    = map[string]bool{"G08": true, "G09": true, "G10": true, "G14": true, "G16": true, "G17": true, "G18": true, "G23": true}
)

// rubricIssues are the failing trigger rules of c as advisory issues.
func rubricIssues(c lecture.Content, ctx quality.Context) []lecture.Issue {
	return quality.Check(c, ctx).IssuesFor(triggerRules)
}

// rubricNotes are the failing note rules of c (LoopFuncs.Notes).
func rubricNotes(c lecture.Content, ctx quality.Context) []lecture.Issue {
	return quality.Check(c, ctx).IssuesFor(noteRules)
}

// RubricScore is one LLM self-assessment score of the review (E01~E13).
type RubricScore struct {
	ID    string `json:"id"`
	Score int    `json:"score"`
	Quote string `json:"quote"`
	Why   string `json:"why,omitempty"`
}

// IndependentAnswer is one problem solved without seeing the draft's answer.
type IndependentAnswer struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Answer string `json:"answer"`
	Stated string `json:"stated"`
}

var rubricIDs = func() map[string]bool {
	m := map[string]bool{}
	for _, it := range prompts.RubricItems {
		m[it.ID] = true
	}
	return m
}()

// validRubric keeps scores with a known id, a score of 1~4 and a quote found
// in the reviewed markup (whitespace-normalized); the first score per id wins.
func validRubric(scores []RubricScore, markup string) []RubricScore {
	text := squash(markup)
	seen := map[string]bool{}
	out := []RubricScore{}
	for _, s := range scores {
		q := squash(s.Quote)
		if !rubricIDs[s.ID] || seen[s.ID] || s.Score < 1 || s.Score > 4 || utf8.RuneCountInString(q) < 4 || !strings.Contains(text, q) {
			continue
		}
		seen[s.ID] = true
		out = append(out, s)
	}
	return out
}

func squash(s string) string { return strings.Join(strings.Fields(s), " ") }

// LectureQuality is the JSON stored in lecture_reviews.quality.
type LectureQuality struct {
	Version       int                  `json:"version"`
	PromptVersion string               `json:"promptVersion"`
	Machine       quality.Report       `json:"machine"`
	LLM           []RubricScore        `json:"llm"`
	Independent   []IndependentAnswer  `json:"independent,omitempty"`
	Links         []quality.LinkResult `json:"links,omitempty"`
}

// qualityJSON is the stored quality of content and its review outcome.
func qualityJSON(c lecture.Content, ctx quality.Context, promptVersion string, r ReviewOutcome) json.RawMessage {
	q := LectureQuality{Version: quality.ReportVersion, PromptVersion: promptVersion, Machine: quality.Check(c, ctx),
		LLM: r.Rubric, Independent: r.Independent, Links: r.Links}
	if q.LLM == nil {
		q.LLM = []RubricScore{}
	}
	raw, err := json.Marshal(q)
	if err != nil {
		logf("encode lecture quality: %v", err)
		return json.RawMessage("{}")
	}
	return raw
}

// maxEvidenceLines bounds each evidence list of the review prompt.
const maxEvidenceLines = 6

// reviewEvidence computes what the review gets besides the draft: the SQL
// runs and recomputed arithmetic of the rubric, the independent answers
// (LectureVerify) and the broken links (Links). Failures of the optional
// calls only log — the review goes on without them.
func (s *Service) reviewEvidence(ctx context.Context, apiKey string, t Target, pc prompts.LectureContext, c lecture.Content, u *usage) (prompts.ReviewEvidence, []IndependentAnswer, []quality.LinkResult) {
	var ev prompts.ReviewEvidence
	rep := quality.Check(c, t.Quality)
	for _, r := range rep.SQL {
		if (r.Status == quality.SQLMismatch || r.Status == quality.SQLRan) && len(ev.SQL) < maxEvidenceLines {
			ev.SQL = append(ev.SQL, fmt.Sprintf("[%s] %s %s — 실행 결과: %s", r.Anchor, sqlStatusKo[r.Status], r.Reason, r.Output))
		}
	}
	for _, a := range rep.Arithmetic {
		if len(ev.Arithmetic) < maxEvidenceLines {
			ev.Arithmetic = append(ev.Arithmetic, fmt.Sprintf("[%s] %s = %s 라고 썼지만 계산하면 %s", a.Anchor, a.Expr, a.Stated, a.Computed))
		}
	}
	vctx, cancel := context.WithTimeout(ctx, verifyTimeout)
	defer cancel()
	linksDone := make(chan []quality.LinkResult, 1)
	go func() { linksDone <- s.Links.Check(vctx, c, t.Quality.AllowedHosts) }()
	var answers []IndependentAnswer
	if s.Cfg.LectureVerify {
		var err error
		if answers, err = s.solveIndependently(vctx, apiKey, pc.ConceptName, c, u); err != nil {
			logf("independent solving of %s: %v (review goes on without it)", t.Plan.ConceptID, err)
		}
	}
	for _, a := range answers {
		if len(ev.Independent) < maxEvidenceLines {
			ev.Independent = append(ev.Independent, fmt.Sprintf("[%s] (%s) 따로 푼 답: %s / 본문: %s", a.ID, a.Kind, a.Answer, a.Stated))
		}
	}
	links := <-linksDone
	for _, l := range links {
		if l.Broken {
			ev.Links = append(ev.Links, fmt.Sprintf("%s (HTTP %d)", l.URL, l.Status))
		}
	}
	return ev, answers, links
}

var sqlStatusKo = map[string]string{quality.SQLMismatch: "불일치:", quality.SQLRan: "실행됨"}

// maxRecomputeItems bounds the independent pass of one draft.
const maxRecomputeItems = 6

// recomputeItems are the draft's exercises (with the code or trace before
// them as context) and pause questions (with the code they ask about), each
// with the draft's own answer kept aside for the comparison.
func recomputeItems(c lecture.Content) ([]prompts.RecomputeItem, map[string]IndependentAnswer) {
	var items []prompts.RecomputeItem
	stated := map[string]IndependentAnswer{}
	for _, s := range c.Sections {
		for i, b := range s.Blocks {
			if len(items) == maxRecomputeItems {
				return items, stated
			}
			var context, answer string
			switch {
			case b.Type == lecture.BlockExercise && i+1 < len(s.Blocks) && s.Blocks[i+1].Type == lecture.BlockAnswer:
				context, answer = previousMaterial(s.Blocks[:i]), s.Blocks[i+1].Text
			case b.Type == lecture.BlockPause && i > 0 && i+1 < len(s.Blocks) && s.Blocks[i-1].Type == lecture.BlockCode:
				context, answer = "```"+s.Blocks[i-1].Lang+"\n"+s.Blocks[i-1].Text+"\n```", s.Blocks[i+1].Text
			default:
				continue
			}
			items = append(items, prompts.RecomputeItem{ID: b.Anchor, Kind: b.Type, Question: b.Text, Context: clipRunes(context, 1500)})
			stated[b.Anchor] = IndependentAnswer{ID: b.Anchor, Kind: b.Type, Stated: clipRunes(squash(answer), 300)}
		}
	}
	return items, stated
}

// previousMaterial is the nearest code or trace before an exercise (its context), "" when none.
func previousMaterial(blocks []lecture.Block) string {
	for i := len(blocks) - 1; i >= 0 && i >= len(blocks)-4; i-- {
		switch b := blocks[i]; b.Type {
		case lecture.BlockCode:
			return "```" + b.Lang + "\n" + b.Text + "\n```"
		case lecture.BlockTrace:
			return string(b.Data)
		}
	}
	return ""
}

func clipRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

type recomputeReply struct {
	Answers []struct {
		ID     string `json:"id"`
		Answer string `json:"answer"`
	} `json:"answers"`
}

// solveIndependently asks the fast model to solve the draft's problems
// without its solutions and outputs (research #11, factored verification).
func (s *Service) solveIndependently(ctx context.Context, apiKey, conceptName string, c lecture.Content, u *usage) ([]IndependentAnswer, error) {
	items, stated := recomputeItems(c)
	if len(items) == 0 {
		return nil, nil
	}
	text, err := s.complete(ctx, apiKey, llm.MessagesRequest{Model: s.Cfg.FastModel, System: prompts.SystemRecompute,
		User: prompts.Recompute(conceptName, items), MaxTokens: 1200, Timeout: verifyTimeout}, u)
	if err != nil {
		return nil, err
	}
	reply, err := llmjson.TryParseJSON[recomputeReply](text)
	if err != nil {
		return nil, fmt.Errorf("parse independent answers: %w", err)
	}
	var out []IndependentAnswer
	for _, a := range reply.Answers {
		st, ok := stated[a.ID]
		if !ok || strings.TrimSpace(a.Answer) == "" {
			continue
		}
		st.Answer = clipRunes(squash(a.Answer), 300)
		out = append(out, st)
	}
	return out, nil
}
