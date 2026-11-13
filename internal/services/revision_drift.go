package services

import (
	"fmt"
	"strings"

	"github.com/wooo-jin/punisher/internal/lecture"
)

// driftWarnRatio: a partial rewrite that changed more than this share of the
// blocks no finding pointed at is logged as a warning (the rewrite was asked
// to keep them verbatim).
const driftWarnRatio = 0.3

// Drift compares a partial rewrite with the draft it revised, block by block:
// Unrelated = blocks of the previous draft no finding pointed at, Changed =
// how many of those are no longer found verbatim in the same section;
// Blocks = blocks of the rewrite, Rewritten = how many of those the previous
// draft did not have verbatim in the same section (what a targeted recheck reads).
type Drift struct {
	Unrelated, Changed int
	Blocks, Rewritten  int
}

// RewrittenRatio is Rewritten / Blocks (1 when the rewrite has no block).
func (d Drift) RewrittenRatio() float64 {
	if d.Blocks == 0 {
		return 1
	}
	return float64(d.Rewritten) / float64(d.Blocks)
}

// Ratio is Changed / Unrelated (0 when nothing was unrelated).
func (d Drift) Ratio() float64 {
	if d.Unrelated == 0 {
		return 0
	}
	return float64(d.Changed) / float64(d.Unrelated)
}

// Warn: the rewrite changed too much outside the findings.
func (d Drift) Warn() bool { return d.Ratio() > driftWarnRatio }

func (d Drift) String() string {
	s := fmt.Sprintf("unrelated blocks changed %d/%d (%.0f%%); rewritten blocks %d/%d (%.0f%%)",
		d.Changed, d.Unrelated, d.Ratio()*100, d.Rewritten, d.Blocks, d.RewrittenRatio()*100)
	if d.Warn() {
		s = "warning: " + s
	}
	return s
}

// measureDrift checks which blocks of prev the findings point at (a block
// path of a validation issue, or a block containing a finding's quote; a
// finding with a section but no matching quote covers its whole section), and
// counts the other blocks that next no longer has verbatim in the same section.
func measureDrift(prev Attempt, next lecture.Content) *Drift {
	flagged := flaggedBlocks(prev)
	have := sectionBlockKeys(next)
	d := &Drift{Rewritten: len(rewrittenBlocks(prev.Content, next))}
	for _, s := range next.Sections {
		d.Blocks += len(s.Blocks)
	}
	for i, s := range prev.Content.Sections {
		for j, b := range s.Blocks {
			if flagged[[2]int{i, j}] {
				continue
			}
			d.Unrelated++
			if !have[s.Kind+"\x00"+blockKey(b)] {
				d.Changed++
			}
		}
	}
	return d
}

// sectionBlockKeys is the set of (section kind, block content) of c.
func sectionBlockKeys(c lecture.Content) map[string]bool {
	have := map[string]bool{}
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			have[s.Kind+"\x00"+blockKey(b)] = true
		}
	}
	return have
}

// rewrittenBlocks are the (section, block) positions of next that prev does
// not have verbatim in the same section: the blocks a rewrite changed or added.
func rewrittenBlocks(prev, next lecture.Content) map[[2]int]bool {
	had := sectionBlockKeys(prev)
	out := map[[2]int]bool{}
	for i, s := range next.Sections {
		for j, b := range s.Blocks {
			if !had[s.Kind+"\x00"+blockKey(b)] {
				out[[2]int{i, j}] = true
			}
		}
	}
	return out
}

// quoteBlocks are the positions of the blocks of c whose text contains the
// quote (its first 40 runes, spaces normalized); none for an empty quote.
func quoteBlocks(c lecture.Content, quote string) [][2]int {
	q := normalizeSpace(quote)
	if q == "" {
		return nil
	}
	if r := []rune(q); len(r) > 40 {
		q = string(r[:40])
	}
	var out [][2]int
	for i, s := range c.Sections {
		for j, b := range s.Blocks {
			if strings.Contains(normalizeSpace(b.Text+" "+string(b.Data)), q) {
				out = append(out, [2]int{i, j})
			}
		}
	}
	return out
}

// blockKey is a block's comparable content (anchors are positional).
func blockKey(b lecture.Block) string {
	src := make([]string, 0, len(b.Sources))
	for _, s := range b.Sources {
		src = append(src, s.Title+" "+s.URL)
	}
	return strings.Join([]string{b.Type, b.Lang, b.Kind, b.Term, b.ConceptID,
		normalizeSpace(b.Text), normalizeSpace(string(b.Data)), strings.Join(src, "\n")}, "\x00")
}

func normalizeSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

// flaggedBlocks are the (section, block) positions of prev the findings point at.
func flaggedBlocks(prev Attempt) map[[2]int]bool {
	out := map[[2]int]bool{}
	secs := prev.Content.Sections
	for _, is := range prev.Issues {
		var i, j int
		if n, _ := fmt.Sscanf(is.Path, "sections[%d].blocks[%d]", &i, &j); n == 2 {
			out[[2]int{i, j}] = true
		}
	}
	type finding struct{ section, quote string }
	var fs []finding
	for _, is := range prev.Review.Issues {
		fs = append(fs, finding{is.Section, is.Quote})
	}
	for _, g := range prev.Review.ReaderGaps {
		fs = append(fs, finding{g.Section, g.Quote})
	}
	for _, f := range fs {
		matched := quoteBlocks(prev.Content, f.quote)
		for _, p := range matched {
			out[p] = true
		}
		if len(matched) > 0 || f.section == "" {
			continue
		}
		for i, s := range secs {
			if s.Kind == f.section {
				for j := range s.Blocks {
					out[[2]int{i, j}] = true
				}
			}
		}
	}
	return out
}
