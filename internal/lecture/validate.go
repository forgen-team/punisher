package lecture

import (
	"fmt"
	"net/url"
	"regexp"
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
	case BlockMarkdown, BlockMermaid, BlockMisconception:
		if strings.TrimSpace(b.Text) == "" {
			add(path, "%s block is empty", b.Type)
		}
		if b.Type != BlockMermaid && UnclosedFence(b.Text) {
			add(path, "%s block has an unclosed ``` code fence (the rest of the text would render as code)", b.Type)
		}
		if b.Type != BlockMermaid {
			if why := MermaidOutsideBlock(b.Text); why != "" {
				add(path, "mermaid diagram inside a %s block (%s) — put the diagram in its own @@mermaid block, without ``` fences", b.Type, why)
			}
		}
		if b.Type == BlockMermaid {
			if why := MermaidSwallowedProse(b.Text); why != "" {
				add(path, "mermaid block swallowed lecture prose (%s) — end the diagram and start a new @@markdown block", why)
			}
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

// mermaidProseRun: this many consecutive prose lines after a blank line means
// the diagram's end was missed and the following text fell into it.
const mermaidProseRun = 2

var (
	hangulRe        = regexp.MustCompile(`[\p{Hangul}]`)
	mdHeadingRe     = regexp.MustCompile(`^#{1,6}\s`)
	mermaidSyntaxRe = regexp.MustCompile(`-->|---|==>|-\.->|->>|-->>|--x|:::|^\s*(flowchart|graph|sequenceDiagram|stateDiagram|classDiagram|erDiagram|gantt|pie|journey|participant|actor|subgraph|end|classDef|class|style|linkStyle|click|note|loop|alt|else|opt|par|rect|autonumber|activate|deactivate|state|direction|section|title|%%)\b|^\s*[\w.-]+\s*[\[\(\{>]`)
)

// MermaidSwallowedProse returns why a mermaid source contains lecture prose
// ("" when it looks like a diagram): a markdown heading (## ), an @@ directive,
// or, after a blank line, mermaidProseRun+ consecutive Korean lines that are
// not diagram syntax.
func MermaidSwallowedProse(src string) string {
	afterBlank, run := false, 0
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case mdHeadingRe.MatchString(t):
			return "markdown heading " + clipLine(t)
		case strings.Contains(t, "@@"):
			return "@@ directive " + clipLine(t)
		case t == "":
			afterBlank, run = true, 0
			continue
		}
		if afterBlank && hangulRe.MatchString(t) && !mermaidSyntaxRe.MatchString(t) {
			run++
			if run >= mermaidProseRun {
				return "prose paragraph " + clipLine(t)
			}
			continue
		}
		afterBlank, run = false, 0
	}
	return ""
}

// mermaidHeaderRe matches a line that is only a mermaid diagram header
// ("sequenceDiagram", "flowchart LR", "graph TD", ...). Prose that merely
// mentions the word does not match because the whole line must be the header.
var mermaidHeaderRe = regexp.MustCompile(`^\s*(sequenceDiagram|classDiagram|erDiagram|gantt|journey|mindmap|timeline|stateDiagram(-v2)?|pie(\s+(showData|title\b.*))?|(flowchart|graph)\s+(TD|TB|BT|RL|LR))\s*$`)

// MermaidOutsideBlock returns why markdown text carries a mermaid diagram
// that the renderer would show as plain code ("" when it does not): a
// ```mermaid fence, a plain fence whose first line is a diagram header, or a
// bare header line outside any fence (2026-11-09: a sequenceDiagram appeared
// as raw code in an instant lecture).
func MermaidOutsideBlock(text string) string {
	open, lang, first := "", "", true
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimLeft(line, " ")
		indented := len(line)-len(t) > 3
		if marker := fenceMarker(t); marker != "" && !indented {
			if open == "" {
				open, lang, first = marker, strings.ToLower(strings.TrimSpace(t[len(marker):])), true
				if lang == "mermaid" {
					return "```mermaid code fence"
				}
				continue
			}
			if marker[0] == open[0] && len(marker) >= len(open) && strings.TrimSpace(t[len(marker):]) == "" {
				open = ""
				continue
			}
		}
		if open != "" {
			if first && strings.TrimSpace(line) != "" {
				first = false
				if (lang == "" || lang == "text") && mermaidHeaderRe.MatchString(line) {
					return "code fence starting with " + clipLine(strings.TrimSpace(line))
				}
			}
			continue
		}
		if mermaidHeaderRe.MatchString(line) {
			return "bare diagram header " + clipLine(strings.TrimSpace(line))
		}
	}
	return ""
}

func clipLine(s string) string {
	r := []rune(s)
	if len(r) > 40 {
		return string(r[:40]) + "…"
	}
	return s
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
