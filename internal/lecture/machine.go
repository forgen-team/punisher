package lecture

import (
	"fmt"
	"regexp"
	"strings"
)

// Prose volume bounds of a new lecture (quality design v2.2 §4: 본문 6,500~11,000자, 공백 제외).
const (
	MinProseChars = 6500
	MaxProseChars = 11000
)

// Minimum material blocks of a new lecture (§4 블록 구성).
const (
	MinExercises = 2
	MinPauses    = 1
)

var (
	// metaPhraseRe: generation meta text that leaked into the lecture (2026-11-13 비평: "MDN이 아닌").
	metaPhraseRe = regexp.MustCompile(`MDN이 아닌|다른 강의에서 다룹`)
	// sectionIDRe: internal section kind identifiers shown to the reader.
	sectionIDRe = regexp.MustCompile(`\b(?:situation_question|principles|walkthrough|judgment_next)\b`)
)

// MachineCheck runs the code checks of a newly generated lecture before the
// review pass (quality design v2.2 §4): self links, meta phrases, the material
// block composition and the prose volume. Structured block data is checked by
// Validate. Stored lectures are never checked with this (old lectures have no
// material blocks and must still render).
//
// Self links and meta phrases block publishing; the composition and volume
// findings are Advisory (teaching quality: a rewrite gets them, publishing
// does not wait on them).
func MachineCheck(c Content) []Issue {
	var issues []Issue
	add := func(path, format string, args ...any) {
		issues = append(issues, Issue{Path: path, Message: fmt.Sprintf(format, args...)})
	}
	advise := func(path, format string, args ...any) {
		issues = append(issues, Issue{Path: path, Message: fmt.Sprintf(format, args...), Advisory: true})
	}
	checkLeaks(c, add)
	checkComposition(c, advise)
	if n := MeasureContent(c).ProseChars; n < MinProseChars || n > MaxProseChars {
		advise("sections", "prose must be %d..%d characters without spaces (code/log excluded), got %d", MinProseChars, MaxProseChars, n)
	}
	return issues
}

// BlockingIssues returns the issues that block publishing (not Advisory).
func BlockingIssues(issues []Issue) []Issue {
	var out []Issue
	for _, i := range issues {
		if !i.Advisory {
			out = append(out, i)
		}
	}
	return out
}

// checkLeaks reports links to the lecture's own concept and meta phrases in
// the reader-visible prose (code, logs and inline code are exempt).
func checkLeaks(c Content, add func(string, string, ...any)) {
	self := "(term:" + c.ConceptID + ")"
	visible := func(path, text string) {
		if c.ConceptID != "" && strings.Contains(text, self) {
			add(path, "links the lecture's own concept (term:%s) — define it in place instead of linking to itself", c.ConceptID)
		}
		prose := inlineCodeRe.ReplaceAllString(ProseOnly(text), "")
		if m := metaPhraseRe.FindString(prose); m != "" {
			add(path, "generation meta phrase %q is visible to the reader — write the content itself", m)
		}
		if m := sectionIDRe.FindString(prose); m != "" {
			add(path, "internal section id %q is visible to the reader — use the section title", m)
		}
	}
	visible("title", c.Title)
	for i, s := range c.Sections {
		path := fmt.Sprintf("sections[%d]", i)
		visible(path+".title", s.Title)
		for j, b := range s.Blocks {
			bpath := fmt.Sprintf("%s.blocks[%d]", path, j)
			if b.Type == BlockTerm && b.ConceptID == c.ConceptID && c.ConceptID != "" {
				add(bpath, "term block points to the lecture's own concept %q", c.ConceptID)
			}
			if markdownBodied[b.Type] || b.Type == BlockPause {
				visible(bpath, b.Text)
			}
		}
	}
}

// checkComposition: principles has a definition keypoint and a trace; the
// lecture has 2+ exercises each followed by its answer, 1+ pause followed by
// the output it hides, and exactly one summary, in judgment_next.
func checkComposition(c Content, add func(string, string, ...any)) {
	exercises, pauses, summaries := 0, 0, 0
	for i, s := range c.Sections {
		definitions, traces := 0, 0
		for j, b := range s.Blocks {
			next := ""
			if j+1 < len(s.Blocks) {
				next = s.Blocks[j+1].Type
			}
			switch b.Type {
			case BlockKeypoint:
				if b.Kind == KeypointDefinition {
					definitions++
				}
			case BlockTrace:
				traces++
			case BlockExercise:
				if next == BlockAnswer {
					exercises++
				}
			case BlockPause:
				if next == BlockLog || next == BlockCode {
					pauses++
				}
			case BlockSummary:
				summaries++
				if s.Kind != SectionJudgment {
					add(fmt.Sprintf("sections[%d]", i), "@@summary belongs in judgment_next, found in %s", s.Kind)
				}
			}
		}
		if s.Kind == SectionPrinciples {
			if definitions == 0 {
				add(fmt.Sprintf("sections[%d]", i), "principles needs at least one @@keypoint kind=definition (the formal definition of the objectives' key noun)")
			}
			if traces == 0 {
				add(fmt.Sprintf("sections[%d]", i), "principles needs at least one @@trace (a hand calculation with small values)")
			}
		}
	}
	if exercises < MinExercises {
		add("sections", "the lecture needs at least %d @@exercise blocks each directly followed by @@answer, got %d", MinExercises, exercises)
	}
	if pauses < MinPauses {
		add("sections", "the lecture needs at least %d @@pause directly followed by the @@log/@@code it asks about", MinPauses)
	}
	if summaries != 1 {
		add("sections", "the lecture needs exactly one @@summary (in judgment_next), got %d", summaries)
	}
}
