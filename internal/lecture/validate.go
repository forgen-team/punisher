package lecture

import (
	"fmt"
	"net/url"
	"strings"
)

// Minutes bounds of the section minutes a new lecture declares (quality
// design v2.2 §3.2: ~20 minutes). The displayed time is learning.EstimateMinutes.
const (
	MinTotalMinutes = 15
	MaxTotalMinutes = 25
)

// ValidationContext carries the facts the validator checks against.
type ValidationContext struct {
	// KnownConceptIDs: concept ids that exist (active or proposed) in the graph.
	KnownConceptIDs map[string]bool
	// PlannedObjectiveIDs: objectives this lecture must cover.
	PlannedObjectiveIDs []string
	// AllowedObjectiveIDs: every objective id of the concept (covered ids must be in this set).
	AllowedObjectiveIDs map[string]bool
}

// Issue is one validation problem.
type Issue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	// Advisory: a teaching-quality finding (material block composition, prose
	// volume) that is fed to a rewrite but never blocks publishing. Every other
	// issue (format, structured block data, self links, meta phrases) blocks.
	Advisory bool `json:"advisory,omitempty"`
}

func (i Issue) String() string { return i.Path + ": " + i.Message }

// Validate checks structural rules. It is pure; an empty result means valid.
//
// Rules: header fields present; exactly the four sections in order with
// positive minutes and at least one non-source block; total minutes 25–35;
// blocks well-formed (code/log need lang, term needs concept id that exists
// or is marked proposed); factual sections (principles, walkthrough) cite at
// least one https source; covered objectives include every planned objective
// and none outside the concept.
func Validate(c Content, vc ValidationContext) []Issue {
	var issues []Issue
	add := func(path, format string, args ...any) {
		issues = append(issues, Issue{Path: path, Message: fmt.Sprintf(format, args...)})
	}

	if strings.TrimSpace(c.Title) == "" {
		add("title", "title is required")
	}
	if c.ConceptID == "" {
		add("conceptId", "conceptId is required")
	}
	if c.Depth < 1 || c.Depth > 3 {
		add("depth", "depth must be 1..3, got %d", c.Depth)
	}
	if len(c.NewScope) == 0 {
		add("newScope", "newScope (what is new in this lecture) is required")
	}
	validateObjectives(c, vc, add)
	validatePrereqRefs(c, vc, add)

	if len(c.Sections) != len(SectionOrder) {
		add("sections", "expected %d sections, got %d", len(SectionOrder), len(c.Sections))
	}
	for i, s := range c.Sections {
		path := fmt.Sprintf("sections[%d]", i)
		if i < len(SectionOrder) && s.Kind != SectionOrder[i] {
			add(path, "expected section %q, got %q", SectionOrder[i], s.Kind)
		}
		validateSection(path, s, vc, add)
	}
	if total := c.TotalMinutes(); total < MinTotalMinutes || total > MaxTotalMinutes {
		add("sections", "total minutes must be %d..%d, got %d", MinTotalMinutes, MaxTotalMinutes, total)
	}
	return issues
}

func validateObjectives(c Content, vc ValidationContext, add func(string, string, ...any)) {
	covered := map[string]bool{}
	for _, id := range c.ObjectiveIDs {
		covered[id] = true
		if vc.AllowedObjectiveIDs != nil && !vc.AllowedObjectiveIDs[id] {
			add("objectiveIds", "objective %q does not belong to concept %q", id, c.ConceptID)
		}
	}
	for _, id := range vc.PlannedObjectiveIDs {
		if !covered[id] {
			add("objectiveIds", "planned objective %q is not covered", id)
		}
	}
	if len(c.ObjectiveIDs) == 0 {
		add("objectiveIds", "at least one covered objective is required")
	}
}

func validatePrereqRefs(c Content, vc ValidationContext, add func(string, string, ...any)) {
	for _, id := range c.PrerequisitesReferenced {
		if vc.KnownConceptIDs != nil && !vc.KnownConceptIDs[id] {
			add("prerequisitesReferenced", "unknown concept %q", id)
		}
	}
}

func validateSection(path string, s Section, vc ValidationContext, add func(string, string, ...any)) {
	if s.Minutes <= 0 {
		add(path, "minutes must be positive")
	}
	if strings.TrimSpace(s.Title) == "" {
		add(path, "title is required")
	}
	contentBlocks, sources := 0, 0
	for j, b := range s.Blocks {
		bpath := fmt.Sprintf("%s.blocks[%d]", path, j)
		validateBlock(bpath, b, vc, add)
		validateSequence(bpath, s.Blocks, j, add)
		if b.Type == BlockSources {
			sources += len(b.Sources)
		} else {
			contentBlocks++
		}
	}
	if contentBlocks == 0 {
		add(path, "section has no content blocks")
	}
	if FactualSections[s.Kind] && sources == 0 {
		add(path, "factual section %q must cite at least one source", s.Kind)
	}
}

// validateSequence checks the pairing rules of §6: @@pause is followed by
// the output it hides (log/code), @@exercise by its @@answer, and an @@answer
// only follows an @@exercise.
func validateSequence(path string, blocks []Block, j int, add func(string, string, ...any)) {
	next := ""
	if j+1 < len(blocks) {
		next = blocks[j+1].Type
	}
	switch blocks[j].Type {
	case BlockPause:
		if next != BlockLog && next != BlockCode {
			add(path, "@@pause must be followed directly by the @@log or @@code it asks about")
		}
	case BlockExercise:
		if next != BlockAnswer {
			add(path, "@@exercise must be followed directly by its @@answer")
		}
	case BlockAnswer:
		if j == 0 || blocks[j-1].Type != BlockExercise {
			add(path, "@@answer must come right after its @@exercise")
		}
	}
}

// validateProse checks a markdown-bodied block: text present, fences closed,
// no mermaid outside @@mermaid, no directive inside the text.
func validateProse(path string, b Block, add func(string, string, ...any)) {
	if strings.TrimSpace(b.Text) == "" {
		add(path, "%s block is empty", b.Type)
	}
	if UnclosedFence(b.Text) {
		add(path, "%s block has an unclosed ``` code fence (the rest of the text would render as code)", b.Type)
	}
	if why := MermaidOutsideBlock(b.Text); why != "" {
		add(path, "mermaid diagram inside a %s block (%s) — put the diagram in its own @@mermaid block, without ``` fences", b.Type, why)
	}
	if d := StrayDirective(b.Text); d != "" {
		add(path, "%s inside the text of a %s block — directives go only at the very start of a line; link a term in a sentence as [용어](term:개념id)", d, b.Type)
	}
}

// ValidateBlock checks one block with the lecture rules (empty text, unclosed
// fences, mermaid placement, code lang, term ids, https sources). Inquiry
// explanations reuse it so they render with the same guarantees.
func ValidateBlock(path string, b Block, vc ValidationContext) []Issue {
	var issues []Issue
	validateBlock(path, b, vc, func(p, format string, args ...any) {
		issues = append(issues, Issue{Path: p, Message: fmt.Sprintf(format, args...)})
	})
	return issues
}

func validateBlock(path string, b Block, vc ValidationContext, add func(string, string, ...any)) {
	switch b.Type {
	case BlockMermaid:
		if strings.TrimSpace(b.Text) == "" {
			add(path, "%s block is empty", b.Type)
		}
		if why := MermaidSwallowedProse(b.Text); why != "" {
			add(path, "mermaid block swallowed lecture prose (%s) — end the diagram and start a new @@markdown block", why)
		}
	case BlockMarkdown, BlockMisconception, BlockExercise, BlockAnswer:
		validateProse(path, b, add)
	case BlockKeypoint:
		if b.Kind != KeypointDefinition && b.Kind != KeypointCaution && b.Kind != KeypointRule {
			add(path, "keypoint kind must be definition|caution|rule, got %q", b.Kind)
		}
		validateProse(path, b, add)
	case BlockSummary:
		validateProse(path, b, add)
		if n := countBullets(b.Text); n < SummaryMinBullet || n > SummaryMaxBullet {
			add(path, "summary needs %d..%d bullets, got %d", SummaryMinBullet, SummaryMaxBullet, n)
		}
	case BlockPause:
		if strings.TrimSpace(b.Text) == "" || strings.Contains(strings.TrimSpace(b.Text), "\n") {
			add(path, "pause block is one question line")
		}
	case BlockTrace, BlockChart, BlockDiff:
		for _, msg := range structuredProblems(b) {
			add(path, "%s", msg)
		}
		if b.Type == BlockDiff && strings.TrimSpace(b.Lang) == "" {
			add(path, "diff block needs lang")
		}
	case BlockCode, BlockLog:
		if strings.TrimSpace(b.Text) == "" {
			add(path, "%s block is empty", b.Type)
		}
		if strings.TrimSpace(b.Lang) == "" {
			add(path, "%s block needs lang", b.Type)
		}
	case BlockTerm:
		if b.ConceptID == "" || strings.TrimSpace(b.Term) == "" {
			add(path, "term block needs concept and term")
		} else if !b.Proposed && vc.KnownConceptIDs != nil && !vc.KnownConceptIDs[b.ConceptID] {
			add(path, "term concept %q does not exist and is not marked proposed", b.ConceptID)
		}
	case BlockSources:
		if len(b.Sources) == 0 {
			add(path, "sources block has no entries")
		}
		for _, src := range b.Sources {
			if !isHTTPSURL(src.URL) {
				add(path, "source url must be https: %q", src.URL)
			}
		}
	default:
		add(path, "unknown block type %q", b.Type)
	}
}

func isHTTPSURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != ""
}

// ProposedTerms returns term blocks marked proposed (concepts to add to the graph).
func ProposedTerms(c Content) []Block {
	out := []Block{}
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			if b.Type == BlockTerm && b.Proposed {
				out = append(out, b)
			}
		}
	}
	return out
}
