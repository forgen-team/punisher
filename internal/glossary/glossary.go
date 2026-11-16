// Package glossary is the shipped term dictionary (curriculum/glossary/*.yaml):
// a fixed popover gloss for every knowledge-map concept, and basic terms that
// are not concepts (SYN, WAL, 시그널 …) which lectures link as term:g.<id>
// instead of re-defining them in every lecture.
package glossary

import (
	"bytes"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// TermPrefix marks a glossary term id (term:g.syn) apart from concept ids.
const TermPrefix = "g."

// MaxGlossRunes bounds one gloss (popover size).
const MaxGlossRunes = 260

// Term is a basic term of the dictionary.
type Term struct {
	ID      string   `yaml:"id" json:"id"`
	Name    string   `yaml:"name" json:"name"`
	Aliases []string `yaml:"aliases" json:"aliases"`
	// Concept: the closest knowledge-map concept ("" = none), for '더 깊게'.
	Concept string `yaml:"concept" json:"concept"`
	Gloss   string `yaml:"gloss" json:"gloss"`
	Area    string `yaml:"-" json:"area"`
}

// Glossary holds the concept glosses and terms.
type Glossary struct {
	concepts map[string]string
	terms    map[string]Term
	order    []string
}

type file struct {
	Area     string            `yaml:"area"`
	Concepts map[string]string `yaml:"concepts"`
	Terms    []Term            `yaml:"terms"`
}

// Load reads glossary/*.yaml from fsys. A missing folder is an empty glossary.
func Load(fsys fs.FS) (*Glossary, error) {
	g := &Glossary{concepts: map[string]string{}, terms: map[string]Term{}}
	names, err := fs.Glob(fsys, "glossary/*.yaml")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	for _, name := range names {
		raw, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		dec := yaml.NewDecoder(bytes.NewReader(raw))
		dec.KnownFields(true)
		var f file
		if err := dec.Decode(&f); err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		for id, text := range f.Concepts {
			if _, dup := g.concepts[id]; dup {
				return nil, fmt.Errorf("%s: concept %s glossed twice", name, id)
			}
			g.concepts[id] = strings.TrimSpace(text)
		}
		for _, t := range f.Terms {
			t.Area, t.Gloss, t.Name = f.Area, strings.TrimSpace(t.Gloss), strings.TrimSpace(t.Name)
			if _, dup := g.terms[t.ID]; dup {
				return nil, fmt.Errorf("%s: term %s defined twice", name, t.ID)
			}
			g.terms[t.ID] = t
			g.order = append(g.order, t.ID)
		}
	}
	sort.Strings(g.order)
	return g, nil
}

// Validate checks the glossary against the knowledge map: concept glosses and
// related concepts exist, term ids are g.<lowercase-hyphen>, names and aliases
// are unique and do not shadow a concept name or alias, glosses fit.
// conceptNames maps a normalized concept name/alias to its id.
func (g *Glossary) Validate(conceptIDs map[string]bool, conceptNames map[string]string) []string {
	var p []string
	for id, text := range g.concepts {
		if !conceptIDs[id] {
			p = append(p, fmt.Sprintf("concept gloss for unknown concept %q", id))
		}
		p = append(p, checkGloss("concept "+id, text)...)
	}
	seen := map[string]string{}
	for _, id := range g.order {
		t := g.terms[id]
		if !validTermID(t.ID) {
			p = append(p, fmt.Sprintf("term id %q: want g.<lowercase-hyphen>", t.ID))
		}
		if t.Name == "" {
			p = append(p, fmt.Sprintf("term %s: name is empty", t.ID))
		}
		if t.Concept != "" && !conceptIDs[t.Concept] {
			p = append(p, fmt.Sprintf("term %s: unknown concept %q", t.ID, t.Concept))
		}
		p = append(p, checkGloss("term "+t.ID, t.Gloss)...)
		for _, n := range append([]string{t.Name}, t.Aliases...) {
			key := Normalize(n)
			if key == "" {
				continue
			}
			if other, ok := seen[key]; ok && other != t.ID {
				p = append(p, fmt.Sprintf("term %s: name %q also used by %s", t.ID, n, other))
			}
			if c, ok := conceptNames[key]; ok {
				p = append(p, fmt.Sprintf("term %s: name %q is concept %s (link the concept instead)", t.ID, n, c))
			}
			seen[key] = t.ID
		}
	}
	sort.Strings(p)
	return p
}

func checkGloss(label, text string) []string {
	switch n := utf8.RuneCountInString(text); {
	case n == 0:
		return []string{label + ": gloss is empty"}
	case n > MaxGlossRunes:
		return []string{fmt.Sprintf("%s: gloss is %d characters (max %d)", label, n, MaxGlossRunes)}
	}
	return nil
}

func validTermID(id string) bool {
	if !strings.HasPrefix(id, TermPrefix) || len(id) <= len(TermPrefix) {
		return false
	}
	for _, r := range id[len(TermPrefix):] {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

// Normalize is the comparison key of a name: lowercase, no spaces.
func Normalize(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), ""))
}

// ConceptGloss is the shipped gloss of a concept ("" = none).
func (g *Glossary) ConceptGloss(id string) string {
	if g == nil {
		return ""
	}
	return g.concepts[id]
}

// Term returns a basic term by id.
func (g *Glossary) Term(id string) (Term, bool) {
	if g == nil {
		return Term{}, false
	}
	t, ok := g.terms[id]
	return t, ok
}

// IsTermID reports a glossary term id (term:g.…).
func IsTermID(id string) bool { return strings.HasPrefix(id, TermPrefix) }

// Terms lists the terms in id order, optionally only those of the given areas
// (plus terms without area).
func (g *Glossary) Terms(areas ...string) []Term {
	if g == nil {
		return nil
	}
	want := map[string]bool{}
	for _, a := range areas {
		want[a] = true
	}
	out := make([]Term, 0, len(g.order))
	for _, id := range g.order {
		t := g.terms[id]
		if len(want) == 0 || want[t.Area] || t.Area == "" {
			out = append(out, t)
		}
	}
	return out
}

// IDs returns every term id (for lecture link validation).
func (g *Glossary) IDs() []string {
	if g == nil {
		return nil
	}
	return append([]string{}, g.order...)
}
