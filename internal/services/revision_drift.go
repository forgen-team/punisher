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
// how many of those are no longer found verbatim in the same section.
type Drift struct {
	Unrelated, Changed int
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
	s := fmt.Sprintf("unrelated blocks changed %d/%d (%.0f%%)", d.Changed, d.Unrelated, d.Ratio()*100)
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
	have := map[string]bool{}
	for _, s := range next.Sections {
		for _, b := range s.Blocks {
			have[s.Kind+"\x00"+blockKey(b)] = true
		}
	}
	d := &Drift{}
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
		q := normalizeSpace(f.quote)
		matched := false
		if q != "" {
			if r := []rune(q); len(r) > 40 {
				q = string(r[:40])
			}
			for i, s := range secs {
				for j, b := range s.Blocks {
					if strings.Contains(normalizeSpace(b.Text+" "+string(b.Data)), q) {
						out[[2]int{i, j}] = true
						matched = true
					}
				}
			}
		}
		if matched || f.section == "" {
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
