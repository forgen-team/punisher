package lecture

import (
	"fmt"
	"net/url"
	"strings"
)

// Minutes bounds for a whole lecture (design §3: ~30 minutes).
const (
	MinTotalMinutes = 25
	MaxTotalMinutes = 35
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

func validateBlock(path string, b Block, vc ValidationContext, add func(string, string, ...any)) {
	switch b.Type {
	case BlockMarkdown, BlockMermaid, BlockMisconception:
		if strings.TrimSpace(b.Text) == "" {
			add(path, "%s block is empty", b.Type)
		}
		if b.Type != BlockMermaid && UnclosedFence(b.Text) {
			add(path, "%s block has an unclosed ``` code fence (the rest of the text would render as code)", b.Type)
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

// UnclosedFence reports whether markdown text opens a ``` / ~~~ code fence it
// never closes (CommonMark: a fence closes with the same character, at least
// as long, on its own line).
func UnclosedFence(text string) bool {
	open := ""
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimLeft(line, " ")
		if len(line)-len(t) > 3 {
			continue
		}
		marker := fenceMarker(t)
		if marker == "" {
			continue
		}
		if open == "" {
			open = marker
		} else if marker[0] == open[0] && len(marker) >= len(open) && strings.TrimSpace(t[len(marker):]) == "" {
			open = ""
		}
	}
	return open != ""
}

func fenceMarker(line string) string {
	for _, ch := range []byte{'`', '~'} {
		n := 0
		for n < len(line) && line[n] == ch {
			n++
		}
		if n >= 3 {
			return line[:n]
		}
	}
	return ""
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
