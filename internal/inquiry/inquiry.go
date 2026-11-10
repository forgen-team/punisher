// Package inquiry parses and validates "이 질문에 답하기" explanations: a
// lecture-style answer to one free-form question, written in the lecture
// markup (same blocks, same renderer) with its own section plan, and the
// follow-up answers of its thread.
package inquiry

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wooo-jin/punisher/internal/curriculum"
	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/models"
)

// Section kinds, in order. assumptions separates guesses about the asker's
// environment from general knowledge (causes cites sources).
const (
	SectionSummary     = "summary"
	SectionCauses      = "causes"
	SectionAssumptions = "assumptions"
	SectionVerify      = "verify"
	SectionFix         = "fix"
	// SectionAnswer is the single section of a follow-up answer as written by the model.
	SectionAnswer = "answer"
	// SectionFollowup is the stored kind of a follow-up section.
	SectionFollowup = "followup"
)

// SectionOrder is the required section plan.
var SectionOrder = []string{SectionSummary, SectionCauses, SectionAssumptions, SectionVerify, SectionFix}

const (
	// MaxNewConcepts bounds concepts proposed for the map per answer.
	MaxNewConcepts = 3
	// MaxRelatedConcepts bounds the nodes one answer attaches to (every inline
	// link would put the question on a dozen nodes — 2026-11-10 first run linked 13).
	MaxRelatedConcepts = 6
	// MaxFollowupRelated / MaxFollowupNew bound what one follow-up adds to the thread.
	MaxFollowupRelated = 3
	MaxFollowupNew     = 2
	// MaxThreadConcepts bounds the nodes a whole thread is attached to.
	MaxThreadConcepts = 12
)

// Answer is a parsed explanation.
type Answer struct {
	Content lecture.Content
	// ConceptIDs: existing map concepts the answer is about (the inquiry is shown on
	// these nodes) — the "concepts:" line, or the first inline links when it is
	// missing; known ids only, capped.
	ConceptIDs []string
	// NewConcepts: concepts not on the map ("new_concept:" lines), sanitized.
	NewConcepts []models.ProposedConcept
}

var (
	conceptsLineRe   = regexp.MustCompile(`(?m)^concepts:\s*(.*)$`)
	newConceptLineRe = regexp.MustCompile(`(?m)^new_concept:\s*(.+)$`)
	slugRe           = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*\.[a-z0-9]+(-[a-z0-9]+)*$`)
	termLinkRe       = regexp.MustCompile(`\[([^\]\n]+)\]\(term:[^)\s]*\)`)
)

// Parse reads a first explanation. known: concept ids the user can see on the map.
func Parse(text string, known map[string]bool) (Answer, error) {
	return parse(text, known, MaxRelatedConcepts, MaxNewConcepts)
}

// ParseFollowup reads a follow-up answer (same markup, one "answer" section).
func ParseFollowup(text string, known map[string]bool) (Answer, error) {
	return parse(text, known, MaxFollowupRelated, MaxFollowupNew)
}

func parse(text string, known map[string]bool, maxRelated, maxNew int) (Answer, error) {
	c, err := lecture.ParseMarkup(text, "", 0)
	if err != nil {
		return Answer{Content: c}, err
	}
	head := text
	if i := strings.Index(text, "@@section"); i >= 0 {
		head = text[:i]
	}
	a := Answer{Content: c}
	a.NewConcepts = parseNewConcepts(head, known, maxNew)
	seen := map[string]bool{}
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id != "" && known[id] && !seen[id] && len(a.ConceptIDs) < maxRelated {
			seen[id] = true
			a.ConceptIDs = append(a.ConceptIDs, id)
		}
	}
	if m := conceptsLineRe.FindStringSubmatch(head); m != nil {
		for _, id := range strings.Split(m[1], ",") {
			add(id)
		}
	}
	if len(a.ConceptIDs) == 0 {
		for _, id := range lecture.InlineTermLinks(c) {
			add(id)
		}
	}
	// 지도에 없는 개념 링크는 글자로 바꾼다(새로 제안한 개념은 저장 성공 뒤 Relink 에서 확정).
	linkable := withNew(known, a.NewConcepts)
	lecture.NormalizeInlineTerms(&a.Content, linkable)
	return a, nil
}

func withNew(known map[string]bool, proposed []models.ProposedConcept) map[string]bool {
	out := make(map[string]bool, len(known)+len(proposed))
	for id := range known {
		out[id] = true
	}
	for _, p := range proposed {
		out[p.ID] = true
	}
	return out
}

// KnownWith returns known plus the proposed concept ids (validation context).
func KnownWith(known map[string]bool, proposed []models.ProposedConcept) map[string]bool {
	return withNew(known, proposed)
}

// parseNewConcepts reads "new_concept: slug | 이름 | 영역 | 선행 개념 id | 한 줄 요약" lines.
// Invalid slugs/areas, existing ids and unknown prerequisites are dropped.
func parseNewConcepts(head string, known map[string]bool, limit int) []models.ProposedConcept {
	var out []models.ProposedConcept
	seen := map[string]bool{}
	for _, m := range newConceptLineRe.FindAllStringSubmatch(head, -1) {
		parts := strings.Split(m[1], "|")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if len(parts) < 3 {
			continue
		}
		p := models.ProposedConcept{ID: strings.ToLower(parts[0]), Name: parts[1], Area: parts[2], Depth: 2}
		if !slugRe.MatchString(p.ID) || known[p.ID] || seen[p.ID] || p.Name == "" || !curriculum.IsValidArea(p.Area) {
			continue
		}
		if len(parts) > 3 && known[parts[3]] {
			p.Prerequisites = []string{parts[3]}
		}
		if len(parts) > 4 {
			p.Summary = parts[4]
		}
		p.Objectives = []string{p.Name + "이(가) 무엇이고 어떤 상황에서 문제를 만드는지 설명한다"}
		seen[p.ID] = true
		out = append(out, p)
		if len(out) == limit {
			break
		}
	}
	return out
}

func validateBlocks(path string, s lecture.Section, vc lecture.ValidationContext, add func(string, ...any)) (content, sources int) {
	for j, b := range s.Blocks {
		for _, issue := range lecture.ValidateBlock(fmt.Sprintf("%s.blocks[%d]", path, j), b, vc) {
			add("%s", issue.String())
		}
		if b.Type == lecture.BlockSources {
			sources += len(b.Sources)
		} else {
			content++
		}
	}
	if content == 0 {
		add("%s: section has no content blocks", path)
	}
	return content, sources
}

// Validate returns problems that make the answer unusable (empty = valid).
func Validate(a Answer, vc lecture.ValidationContext) []string {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	c := a.Content
	if strings.TrimSpace(c.Title) == "" {
		add("title is required")
	}
	if len(c.Sections) != len(SectionOrder) {
		add("expected %d sections (%s), got %d", len(SectionOrder), strings.Join(SectionOrder, ", "), len(c.Sections))
	}
	for i, s := range c.Sections {
		path := fmt.Sprintf("sections[%d]", i)
		if i < len(SectionOrder) && s.Kind != SectionOrder[i] {
			add("%s: expected section %q, got %q", path, SectionOrder[i], s.Kind)
		}
		if strings.TrimSpace(s.Title) == "" {
			add("%s: title is required", path)
		}
		if _, sources := validateBlocks(path, s, vc, add); s.Kind == SectionCauses && sources == 0 {
			add("%s: causes must cite at least one https source (근거)", path)
		}
	}
	return problems
}

// ValidateFollowup returns problems of a follow-up answer (empty = valid).
func ValidateFollowup(a Answer, vc lecture.ValidationContext) []string {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	if len(a.Content.Sections) != 1 || a.Content.Sections[0].Kind != SectionAnswer {
		add("expected exactly one section %q", SectionAnswer)
		return problems
	}
	s := a.Content.Sections[0]
	if strings.TrimSpace(s.Title) == "" {
		add("sections[0]: title is required")
	}
	validateBlocks("sections[0]", s, vc, add)
	return problems
}

// FollowupSection turns a validated answer into the stored section: unique
// anchors (the thread is one document) and the follow-up kind.
func FollowupSection(a Answer, followupID string) lecture.Section {
	s := a.Content.Sections[0]
	short := strings.ReplaceAll(followupID, "-", "")
	if len(short) > 8 {
		short = short[:8]
	}
	s.Kind, s.Anchor, s.Minutes = SectionFollowup, "fu-"+short, 0
	for i := range s.Blocks {
		s.Blocks[i].Anchor = fmt.Sprintf("%s-%d", s.Anchor, i+1)
	}
	return s
}

// Relink turns inline links to concepts outside allowed into plain text (a
// proposed concept whose save failed must not leave a dead link).
func Relink(c *lecture.Content, allowed map[string]bool) {
	lecture.NormalizeInlineTerms(c, allowed)
}

// SectionText is a section as plain context text for a follow-up prompt:
// markdown and code, term links reduced to their labels, no diagrams or sources.
func SectionText(s lecture.Section) string {
	var b strings.Builder
	b.WriteString("[" + s.Title + "]\n")
	for _, block := range s.Blocks {
		switch block.Type {
		case lecture.BlockMarkdown, lecture.BlockMisconception, lecture.BlockCode, lecture.BlockLog:
			b.WriteString(termLinkRe.ReplaceAllString(block.Text, "$1"))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// MergeIDs appends add to base without duplicates, keeping order, up to limit.
func MergeIDs(base, add []string, limit int) []string {
	out := make([]string, 0, len(base)+len(add))
	seen := map[string]bool{}
	for _, id := range append(append([]string{}, base...), add...) {
		if id != "" && !seen[id] && len(out) < limit {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
