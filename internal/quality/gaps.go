package quality

import (
	"strings"

	"github.com/forgen-team/punisher/internal/lecture"
)

// Reader-gap agreement (research #15): does the reviewer's reader simulation
// (reader_gaps) find the blocks where the learner actually needed help —
// questions and passage explanations asked there, or term popovers opened
// from there? Compared per block and per section.

// ReaderGap is a reviewer reader gap (its section and quoted sentence).
type ReaderGap struct {
	Section string `json:"section"`
	Quote   string `json:"quote"`
}

// GapAgreement is the comparison on one lecture.
type GapAgreement struct {
	// GapBlocks: blocks a reader gap quotes; Unplaced: gaps whose quote is in no block.
	GapBlocks int `json:"gapBlocks"`
	Unplaced  int `json:"unplaced"`
	// LearnerBlocks: blocks with a learner signal (questions + term opens ≥ minSignal).
	LearnerBlocks int `json:"learnerBlocks"`
	Both          int `json:"both"`
	// Precision: share of gap blocks the learner also stumbled on; Missed: share
	// of learner blocks no gap pointed at (the reader simulation's misses). -1 = not measurable.
	Precision float64 `json:"precision"`
	Missed    float64 `json:"missed"`
	// Section-level versions of Both / Missed (a gap and a signal in the same section agree).
	SectionBoth   int     `json:"sectionBoth"`
	SectionMissed float64 `json:"sectionMissed"`
}

// CompareGaps places each gap's quote in the block that holds it and compares
// with the blocks whose learner signals (by anchor) reach minSignal.
func CompareGaps(c lecture.Content, gaps []ReaderGap, signals map[string]int, minSignal int) GapAgreement {
	type place struct{ section string }
	blocks := map[string]place{}
	gapBlocks, gapSections := map[string]bool{}, map[string]bool{}
	var a GapAgreement
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			blocks[b.Anchor] = place{s.Kind}
		}
	}
	for _, g := range gaps {
		q := strings.Join(strings.Fields(g.Quote), " ")
		found := ""
		for _, s := range c.Sections {
			for _, b := range s.Blocks {
				if found == "" && q != "" && strings.Contains(strings.Join(strings.Fields(b.Text), " "), q) {
					found = b.Anchor
					gapSections[s.Kind] = true
				}
			}
		}
		if found == "" {
			a.Unplaced++
			continue
		}
		gapBlocks[found] = true
	}
	learnerSections := map[string]bool{}
	for anchor, n := range signals {
		p, ok := blocks[anchor]
		if !ok || n < minSignal {
			continue
		}
		a.LearnerBlocks++
		learnerSections[p.section] = true
		if gapBlocks[anchor] {
			a.Both++
		}
	}
	a.GapBlocks = len(gapBlocks)
	a.Precision, a.Missed = ratio(a.Both, a.GapBlocks), -1
	if a.LearnerBlocks > 0 {
		a.Missed = 1 - float64(a.Both)/float64(a.LearnerBlocks)
	}
	missedSections := 0
	for sec := range learnerSections {
		if gapSections[sec] {
			a.SectionBoth++
		} else {
			missedSections++
		}
	}
	a.SectionMissed = ratio(missedSections, len(learnerSections))
	return a
}

func ratio(n, d int) float64 {
	if d == 0 {
		return -1
	}
	return float64(n) / float64(d)
}
