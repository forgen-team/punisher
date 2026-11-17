// Package curriculum loads, validates and versions the concept-graph seed
// (apps/punisher-server/curriculum/*.yaml).
package curriculum

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Areas is the fixed set of knowledge areas (design §3.1). Each seed file
// declares exactly one of these.
var Areas = []string{
	"data-structures",
	"computer-architecture",
	"os",
	"network-web",
	"database",
	"distributed",
	"infra-ops",
	"security-design",
}

// AreaTitles 는 영역 표시 이름(각 seed YAML 의 title 과 같다). 사용자 문구에 slug 를 노출하지 않기 위해 쓴다.
var AreaTitles = map[string]string{
	"data-structures":       "자료구조와 알고리즘",
	"computer-architecture": "컴퓨터구조",
	"os":                    "운영체제",
	"network-web":           "네트워크와 웹",
	"database":              "데이터베이스",
	"distributed":           "분산 시스템",
	"infra-ops":             "인프라와 운영",
	"security-design":       "보안과 설계",
}

// AreaTitlesEn are the English area names (i18n/en/*.yaml title).
var AreaTitlesEn = map[string]string{
	"data-structures":       "Data Structures and Algorithms",
	"computer-architecture": "Computer Architecture",
	"os":                    "Operating Systems",
	"network-web":           "Networking and the Web",
	"database":              "Databases",
	"distributed":           "Distributed Systems",
	"infra-ops":             "Infrastructure and Operations",
	"security-design":       "Security and Design",
}

// AreaTitleIn is the area name in the locale ("en" → English, otherwise Korean).
func AreaTitleIn(area, locale string) string {
	if locale == "en" {
		if t, ok := AreaTitlesEn[area]; ok {
			return t
		}
	}
	return AreaTitle(area)
}

// AreaTitle 은 영역 표시 이름을 반환한다(모르는 영역이면 slug 그대로).
func AreaTitle(area string) string {
	if t, ok := AreaTitles[area]; ok {
		return t
	}
	return area
}

// IsValidArea reports whether area is one of Areas.
func IsValidArea(area string) bool {
	for _, a := range Areas {
		if a == area {
			return true
		}
	}
	return false
}

// File is one YAML seed file.
type File struct {
	Area     string    `yaml:"area"`
	Title    string    `yaml:"title"`
	Concepts []Concept `yaml:"concepts"`
}

// DocLink is an official documentation reference.
type DocLink struct {
	Title string `yaml:"title" json:"title"`
	URL   string `yaml:"url" json:"url"`
}

// Objective is a learning objective. In YAML it may be a plain string
// (depth inherits from the concept) or a mapping {text, depth}.
type Objective struct {
	Text  string `json:"text"`
	Depth int    `json:"depth"`
	// TextEn: English overlay (i18n/en); "" = fall back to Text.
	TextEn string `json:"textEn,omitempty"`
}

func (o *Objective) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		o.Text = node.Value
		return nil
	}
	var raw struct {
		Text  string `yaml:"text"`
		Depth int    `yaml:"depth"`
	}
	if err := node.Decode(&raw); err != nil {
		return fmt.Errorf("objective: %w", err)
	}
	o.Text, o.Depth = raw.Text, raw.Depth
	return nil
}

// Concept is a node of the concept graph as written in the seed.
type Concept struct {
	ID             string      `yaml:"id" json:"id"`
	Name           string      `yaml:"name" json:"name"`
	Aliases        []string    `yaml:"aliases" json:"aliases"`
	Area           string      `yaml:"area" json:"area"`
	Depth          int         `yaml:"depth" json:"depth"`
	Summary        string      `yaml:"summary" json:"summary"`
	Objectives     []Objective `yaml:"objectives" json:"objectives"`
	Prerequisites  []string    `yaml:"prerequisites" json:"prerequisites"`
	Related        []string    `yaml:"related" json:"related"`
	PartOf         []string    `yaml:"part_of" json:"partOf"`
	PracticalCases []string    `yaml:"practical_cases" json:"practicalCases"`
	DocLinks       []DocLink   `yaml:"doc_links" json:"docLinks"`
	// English overlay (curriculum/i18n/en, applied by Load); "" / nil = fall back to Korean.
	NameEn           string   `yaml:"-" json:"nameEn,omitempty"`
	SummaryEn        string   `yaml:"-" json:"summaryEn,omitempty"`
	PracticalCasesEn []string `yaml:"-" json:"practicalCasesEn,omitempty"`
}

// ObjectiveID returns the stable id of the n-th (1-based) objective.
// Seed authors must append new objectives instead of reordering.
func ObjectiveID(conceptID string, ordinal int) string {
	return fmt.Sprintf("%s#%d", conceptID, ordinal)
}

// Edge kinds.
const (
	EdgePrerequisite = "prerequisite"
	EdgeRelated      = "related"
	EdgePartOf       = "part_of"
)

// Edge is a directed edge. For prerequisite edges From must be learned before To.
type Edge struct {
	From string
	To   string
	Kind string
}

// Alias is a normalized alias pointing at a concept.
type Alias struct {
	Alias     string
	Display   string
	ConceptID string
}

// Seed is the validated, flattened curriculum.
type Seed struct {
	Version  string
	Concepts []Concept // sorted by id
	Edges    []Edge
	Aliases  []Alias
}
