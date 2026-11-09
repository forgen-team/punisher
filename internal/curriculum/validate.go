package curriculum

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var conceptIDPattern = regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9]+(-[a-z0-9]+)*)+$`)

// Validate checks the whole concept set and returns every problem found
// (joined), or nil. Rules: unique well-formed ids, valid area/depth,
// at least one objective, references exist, aliases unique after
// normalization, and an acyclic prerequisite graph.
func Validate(concepts []Concept) error {
	var errs []error
	byID := make(map[string]*Concept, len(concepts))
	for i := range concepts {
		c := &concepts[i]
		if _, dup := byID[c.ID]; dup {
			errs = append(errs, fmt.Errorf("duplicate concept id %q", c.ID))
			continue
		}
		byID[c.ID] = c
	}

	for i := range concepts {
		errs = append(errs, validateConcept(&concepts[i], byID)...)
	}
	errs = append(errs, validateAliases(concepts)...)

	if cycle := FindPrerequisiteCycle(concepts); cycle != nil {
		errs = append(errs, fmt.Errorf("prerequisite cycle: %s", strings.Join(cycle, " -> ")))
	}
	return errors.Join(errs...)
}

func validateConcept(c *Concept, byID map[string]*Concept) []error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("concept %q: "+format, append([]any{c.ID}, args...)...))
	}

	if !conceptIDPattern.MatchString(c.ID) {
		fail("id must look like 'area.slug' (lowercase, dots, hyphens)")
	}
	if strings.TrimSpace(c.Name) == "" {
		fail("name is required")
	}
	if !IsValidArea(c.Area) {
		fail("unknown area %q", c.Area)
	}
	if c.Depth < 1 || c.Depth > 3 {
		fail("depth must be 1..3, got %d", c.Depth)
	}
	if strings.TrimSpace(c.Summary) == "" {
		fail("summary is required")
	}
	if len(c.Objectives) == 0 {
		fail("at least one objective is required")
	}
	for i, o := range c.Objectives {
		if strings.TrimSpace(o.Text) == "" {
			fail("objective %d text is empty", i+1)
		}
		if o.Depth < 1 || o.Depth > 3 {
			fail("objective %d depth must be 1..3, got %d", i+1, o.Depth)
		}
	}
	for _, ref := range refs(c) {
		if ref.id == c.ID {
			fail("%s references itself", ref.kind)
		} else if _, ok := byID[ref.id]; !ok {
			fail("%s %q does not exist", ref.kind, ref.id)
		}
	}
	for _, l := range c.DocLinks {
		if !strings.HasPrefix(l.URL, "https://") {
			fail("doc link %q must be https", l.URL)
		}
	}
	return errs
}

type ref struct{ kind, id string }

func refs(c *Concept) []ref {
	out := make([]ref, 0, len(c.Prerequisites)+len(c.Related)+len(c.PartOf))
	for _, id := range c.Prerequisites {
		out = append(out, ref{EdgePrerequisite, id})
	}
	for _, id := range c.Related {
		out = append(out, ref{EdgeRelated, id})
	}
	for _, id := range c.PartOf {
		out = append(out, ref{EdgePartOf, id})
	}
	return out
}

// validateAliases enforces that every normalized name/alias maps to exactly
// one concept. Repeats within the same concept are tolerated.
func validateAliases(concepts []Concept) []error {
	owner := make(map[string]string)
	var errs []error
	for _, c := range concepts {
		for _, raw := range append([]string{c.Name}, c.Aliases...) {
			key := NormalizeAlias(raw)
			if key == "" {
				errs = append(errs, fmt.Errorf("concept %q: empty alias", c.ID))
				continue
			}
			if prev, ok := owner[key]; ok && prev != c.ID {
				errs = append(errs, fmt.Errorf("alias %q used by both %q and %q", key, prev, c.ID))
				continue
			}
			owner[key] = c.ID
		}
	}
	return errs
}

// FindPrerequisiteCycle returns one cycle (as a closed path of ids) in the
// prerequisite graph, or nil if it is acyclic. Unknown ids are ignored.
func FindPrerequisiteCycle(concepts []Concept) []string {
	prereqs := make(map[string][]string, len(concepts))
	ids := make([]string, 0, len(concepts))
	for _, c := range concepts {
		prereqs[c.ID] = c.Prerequisites
		ids = append(ids, c.ID)
	}
	sort.Strings(ids)

	const (
		unvisited = iota
		visiting
		done
	)
	color := make(map[string]int, len(ids))
	var stack []string
	var cycle []string

	var visit func(id string) bool
	visit = func(id string) bool {
		color[id] = visiting
		stack = append(stack, id)
		for _, p := range prereqs[id] {
			if _, known := prereqs[p]; !known {
				continue
			}
			switch color[p] {
			case visiting:
				cycle = closeCycle(stack, p)
				return true
			case unvisited:
				if visit(p) {
					return true
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[id] = done
		return false
	}

	for _, id := range ids {
		if color[id] == unvisited && visit(id) {
			return cycle
		}
	}
	return nil
}

func closeCycle(stack []string, start string) []string {
	for i, id := range stack {
		if id == start {
			out := append([]string{}, stack[i:]...)
			return append(out, start)
		}
	}
	return []string{start}
}
