// Package lecture defines the lecture content schema (stored as lectures.content
// JSONB), the line-based markup the LLM writes, and a pure validator.
package lecture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// SchemaVersion is bumped when Content changes incompatibly.
const SchemaVersion = 1

// Section kinds, in the fixed order (20-minute lecture, quality design v2.2 §3.2).
const (
	SectionSituation   = "situation_question" // ~2 min
	SectionPrinciples  = "principles"         // ~9 min
	SectionWalkthrough = "walkthrough"        // ~6 min
	SectionJudgment    = "judgment_next"      // ~3 min
)

// SectionOrder is the required section sequence.
var SectionOrder = []string{SectionSituation, SectionPrinciples, SectionWalkthrough, SectionJudgment}

// TargetMinutes is the nominal length of each section.
var TargetMinutes = map[string]int{
	SectionSituation:   2,
	SectionPrinciples:  9,
	SectionWalkthrough: 6,
	SectionJudgment:    3,
}

// FactualSections must cite sources.
var FactualSections = map[string]bool{SectionPrinciples: true, SectionWalkthrough: true}

// Block types.
const (
	BlockMarkdown      = "markdown"
	BlockMermaid       = "mermaid"
	BlockCode          = "code"
	BlockLog           = "log"
	BlockMisconception = "misconception"
	BlockTerm          = "term"
	BlockSources       = "sources"
	// Lecture material blocks (quality design v2.2 §6). trace/chart/diff carry
	// their JSON body in Block.Data; the others carry markdown/plain Text.
	BlockKeypoint = "keypoint" // Kind: definition | caution | rule
	BlockTrace    = "trace"
	BlockChart    = "chart"
	BlockDiff     = "diff" // Lang: language of before/after
	BlockPause    = "pause"
	BlockExercise = "exercise"
	BlockAnswer   = "answer"
	BlockSummary  = "summary"
	// BlockLab: one read-only command for the reader's own terminal (v3 §4). Kind = os
	// (macos | linux | any); Text = "$ command" line, then 보는 곳:/예상: lines.
	BlockLab = "lab"
)

// Keypoint kinds (@@keypoint kind=…).
const (
	KeypointDefinition = "definition"
	KeypointCaution    = "caution"
	KeypointRule       = "rule"
)

// IsBlockType reports whether t is a known block type.
func IsBlockType(t string) bool {
	switch t {
	case BlockMarkdown, BlockMermaid, BlockCode, BlockLog, BlockMisconception, BlockTerm, BlockSources,
		BlockKeypoint, BlockTrace, BlockChart, BlockDiff, BlockPause, BlockExercise, BlockAnswer, BlockSummary, BlockLab:
		return true
	}
	return false
}

// StructuredBlock reports whether a block type's body is one JSON object (Block.Data).
func StructuredBlock(t string) bool {
	return t == BlockTrace || t == BlockChart || t == BlockDiff
}

// Source is a cited reference (official documentation preferred).
type Source struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// Block is one renderable unit. Fields are used per type:
//   - markdown / misconception / mermaid: Text
//   - code / log: Text + Lang
//   - term: Term (display), ConceptID, Gloss (Text), Proposed (concept not yet in graph)
//   - sources: Sources
//   - lab: Kind (os) + Text ("$ command" line, 보는 곳:, 예상:)
//   - keypoint: Kind + Text (markdown); exercise / answer / summary: Text (markdown); pause: Text (one question line)
//   - trace / chart: Data (JSON object, §6); diff: Data + Lang
type Block struct {
	Anchor    string          `json:"anchor"`
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Lang      string          `json:"lang,omitempty"`
	Kind      string          `json:"kind,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Term      string          `json:"term,omitempty"`
	ConceptID string          `json:"conceptId,omitempty"`
	Proposed  bool            `json:"proposed,omitempty"`
	Sources   []Source        `json:"sources,omitempty"`
}

// Section is one of the four fixed parts.
type Section struct {
	Anchor  string  `json:"anchor"`
	Kind    string  `json:"kind"`
	Title   string  `json:"title"`
	Minutes int     `json:"minutes"`
	Blocks  []Block `json:"blocks"`
}

// Content is the full lecture document.
type Content struct {
	SchemaVersion           int       `json:"schemaVersion"`
	Title                   string    `json:"title"`
	ConceptID               string    `json:"conceptId"`
	ObjectiveIDs            []string  `json:"objectiveIds"`
	Depth                   int       `json:"depth"`
	EstimatedMinutes        int       `json:"estimatedMinutes"`
	NewScope                []string  `json:"newScope"`
	PrerequisitesReferenced []string  `json:"prerequisitesReferenced"`
	Sections                []Section `json:"sections"`
}

// TotalMinutes sums section minutes.
func (c Content) TotalMinutes() int {
	total := 0
	for _, s := range c.Sections {
		total += s.Minutes
	}
	return total
}

// AllSources returns de-duplicated sources across all sections (by URL).
func (c Content) AllSources() []Source {
	seen := map[string]bool{}
	out := []Source{}
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			for _, src := range b.Sources {
				if src.URL == "" || seen[src.URL] {
					continue
				}
				seen[src.URL] = true
				out = append(out, src)
			}
		}
	}
	return out
}

// TermConceptIDs returns term concept ids (proposed ones included).
func (c Content) TermConceptIDs() []string {
	out := []string{}
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			if b.Type == BlockTerm && b.ConceptID != "" {
				out = append(out, b.ConceptID)
			}
		}
	}
	return out
}

// Hash is the SHA-256 of the canonical JSON encoding. Lectures are immutable,
// so a changed hash means a new version.
func Hash(c Content) (string, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// SummaryText is the text of the lecture's @@summary block ("" when it has none).
func (c Content) SummaryText() string {
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			if b.Type == BlockSummary {
				return strings.TrimSpace(b.Text)
			}
		}
	}
	return ""
}
