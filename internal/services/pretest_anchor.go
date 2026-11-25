package services

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"unicode"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/lecture"
)

// An instant lecture's pretest is written while the lecture is (research #2):
// from the objectives only, without anchors. When the lecture is first
// opened, each of those items is pointed to the block that answers it, so the
// result can say where to re-read. Deterministic (no LLM call): the block
// whose words weigh most in the item's question and answer.

// minAnchorScore: the share of the item's weighted words a block must hold to be its anchor.
const minAnchorScore = 0.2

// reanchorPretest points the unanchored pretest items of an instant
// lecture's job to its blocks (first open only). Failure only logs.
func (s *Service) reanchorPretest(ctx context.Context, userID, lectureID string) {
	lec, err := s.Store.GetLecture(ctx, userID, lectureID)
	if err != nil || lec.Kind != learning.JobInstant || lec.GenerationJobID == "" || lec.Progress != nil {
		return
	}
	job, err := s.Store.GetJob(ctx, userID, lec.GenerationJobID)
	if err != nil {
		logf("reanchor pretest of %s: %v", lectureID, err)
		return
	}
	items, err := s.Store.UnanchoredPretestItems(ctx, userID, lec.ObjectiveIDs, job.CreatedAt)
	if err != nil || len(items) == 0 {
		return
	}
	var c lecture.Content
	if err := json.Unmarshal(lec.Content, &c); err != nil {
		return
	}
	idx := newAnchorIndex(c)
	linked := 0
	for _, it := range items {
		text := it.Prompt + "\n" + it.Rubric.Answer + "\n" + strings.Join(it.Rubric.Must, "\n")
		if anchor := idx.best(text); anchor != "" {
			if err := s.Store.SetCheckItemAnchor(ctx, userID, it.ID, anchor); err != nil {
				logf("reanchor pretest item %s: %v", it.ID, err)
				continue
			}
			linked++
		}
	}
	logf("pretest of instant lecture %s: %d/%d items point to the body", lectureID, linked, len(items))
}

// anchorIndex is the lecture's answering blocks as weighted word sets.
type anchorIndex struct {
	anchors []string
	grams   []map[string]bool
	idf     map[string]float64
}

// answeringBlock: blocks that explain (recall keys, answers and the summary would give the answer away).
func answeringBlock(t string) bool {
	switch t {
	case lecture.BlockMarkdown, lecture.BlockKeypoint, lecture.BlockMisconception, lecture.BlockTrace,
		lecture.BlockCode, lecture.BlockLog, lecture.BlockDiff, lecture.BlockChart:
		return true
	}
	return false
}

func newAnchorIndex(c lecture.Content) anchorIndex {
	idx := anchorIndex{idf: map[string]float64{}}
	df := map[string]int{}
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			if !answeringBlock(b.Type) || b.Anchor == "" {
				continue
			}
			g := wordGrams(b.Text + " " + string(b.Data))
			idx.anchors = append(idx.anchors, b.Anchor)
			idx.grams = append(idx.grams, g)
			for w := range g {
				df[w]++
			}
		}
	}
	n := float64(len(idx.anchors))
	for w, d := range df {
		idx.idf[w] = math.Log(1 + n/float64(d))
	}
	return idx
}

// best is the anchor holding the largest weighted share of text's words ("" under minAnchorScore).
func (idx anchorIndex) best(text string) string {
	want := wordGrams(text)
	total := 0.0
	for w := range want {
		total += idx.idf[w]
	}
	if total == 0 {
		return ""
	}
	bestAnchor, bestScore := "", minAnchorScore
	for i, g := range idx.grams {
		score := 0.0
		for w := range want {
			if g[w] {
				score += idx.idf[w]
			}
		}
		if score/total > bestScore {
			bestAnchor, bestScore = idx.anchors[i], score/total
		}
	}
	return bestAnchor
}

// wordGrams: lowercase Latin/number words of 2+ characters and Hangul
// syllable bigrams (Korean particles attach to words, so whole words rarely match).
func wordGrams(text string) map[string]bool {
	out := map[string]bool{}
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	for _, f := range fields {
		r := []rune(f)
		if unicode.Is(unicode.Hangul, r[0]) {
			for i := 0; i+1 < len(r); i++ {
				out[string(r[i:i+2])] = true
			}
			continue
		}
		if len(r) >= 2 {
			out[f] = true
		}
	}
	return out
}
