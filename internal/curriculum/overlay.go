package curriculum

import (
	"bytes"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// OverlayDir holds the English overlay files (i18n/en/<area>.yaml): display and
// prompt text only. Ids, objective order, depths and edges come from the
// Korean seed, which stays the canonical source (mastery, change detection).
const OverlayDir = "i18n/en"

type overlayConcept struct {
	Name           string   `yaml:"name"`
	Summary        string   `yaml:"summary"`
	Objectives     []string `yaml:"objectives"`
	PracticalCases []string `yaml:"practical_cases"`
}

type overlayFile struct {
	Area     string                    `yaml:"area"`
	Title    string                    `yaml:"title"`
	Concepts map[string]overlayConcept `yaml:"concepts"`
}

// loadOverlay reads OverlayDir; a missing folder is no overlay.
func loadOverlay(fsys fs.FS) (map[string]overlayConcept, error) {
	names, err := fs.Glob(fsys, OverlayDir+"/*.yaml")
	if err != nil {
		return nil, fmt.Errorf("list overlay files: %w", err)
	}
	sort.Strings(names)
	out := map[string]overlayConcept{}
	for _, name := range names {
		raw, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		dec := yaml.NewDecoder(bytes.NewReader(raw))
		dec.KnownFields(true)
		var f overlayFile
		if err := dec.Decode(&f); err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		for id, c := range f.Concepts {
			if _, dup := out[id]; dup {
				return nil, fmt.Errorf("%s: concept %s translated twice", name, id)
			}
			out[id] = c
		}
	}
	return out, nil
}

// applyOverlay copies the English text onto the concepts. An overlay entry for
// an unknown concept or with a different objective count is an error (the
// overlay must follow the seed); a concept without an entry keeps Korean only.
func applyOverlay(concepts []Concept, overlay map[string]overlayConcept) error {
	known := make(map[string]bool, len(concepts))
	var errs []string
	for i := range concepts {
		c := &concepts[i]
		known[c.ID] = true
		o, ok := overlay[c.ID]
		if !ok {
			continue
		}
		if len(o.Objectives) != len(c.Objectives) {
			errs = append(errs, fmt.Sprintf("%s: %d English objectives for %d", c.ID, len(o.Objectives), len(c.Objectives)))
			continue
		}
		if len(o.PracticalCases) > 0 && len(o.PracticalCases) != len(c.PracticalCases) {
			errs = append(errs, fmt.Sprintf("%s: %d English practical cases for %d", c.ID, len(o.PracticalCases), len(c.PracticalCases)))
			continue
		}
		c.NameEn, c.SummaryEn = strings.TrimSpace(o.Name), strings.TrimSpace(o.Summary)
		c.PracticalCasesEn = o.PracticalCases
		for j := range c.Objectives {
			c.Objectives[j].TextEn = strings.TrimSpace(o.Objectives[j])
		}
	}
	for id := range overlay {
		if !known[id] {
			errs = append(errs, fmt.Sprintf("overlay concept %s is not in the seed", id))
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("english overlay: %s", strings.Join(errs, "; "))
	}
	return nil
}

// NameIn is the concept name in the locale (English falls back to Korean).
func (c Concept) NameIn(locale string) string {
	if locale == "en" && c.NameEn != "" {
		return c.NameEn
	}
	return c.Name
}

// SummaryIn is the concept summary in the locale.
func (c Concept) SummaryIn(locale string) string {
	if locale == "en" && c.SummaryEn != "" {
		return c.SummaryEn
	}
	return c.Summary
}

// TextIn is the objective text in the locale.
func (o Objective) TextIn(locale string) string {
	if locale == "en" && o.TextEn != "" {
		return o.TextEn
	}
	return o.Text
}
