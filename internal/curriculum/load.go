package curriculum

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads every *.yaml file at the root of fsys, validates the combined
// concept set and returns the flattened seed with its content version.
func Load(fsys fs.FS) (*Seed, error) {
	names, err := fs.Glob(fsys, "*.yaml")
	if err != nil {
		return nil, fmt.Errorf("list curriculum files: %w", err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no curriculum *.yaml files found")
	}
	sort.Strings(names)

	var concepts []Concept
	for _, name := range names {
		file, err := decodeFile(fsys, name)
		if err != nil {
			return nil, err
		}
		concepts = append(concepts, file.Concepts...)
	}
	return Build(concepts)
}

func decodeFile(fsys fs.FS, name string) (*File, error) {
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	var file File
	if err := dec.Decode(&file); err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	if !IsValidArea(file.Area) {
		return nil, fmt.Errorf("%s: unknown area %q", name, file.Area)
	}
	for i := range file.Concepts {
		c := &file.Concepts[i]
		if c.Area == "" {
			c.Area = file.Area
		}
		if c.Area != file.Area {
			return nil, fmt.Errorf("%s: concept %q declares area %q, file area is %q", name, c.ID, c.Area, file.Area)
		}
	}
	return &file, nil
}

// Build normalizes, validates and versions an in-memory concept set.
func Build(concepts []Concept) (*Seed, error) {
	for i := range concepts {
		normalizeConcept(&concepts[i])
	}
	if err := Validate(concepts); err != nil {
		return nil, fmt.Errorf("invalid curriculum: %w", err)
	}
	sort.Slice(concepts, func(i, j int) bool { return concepts[i].ID < concepts[j].ID })

	version, err := computeVersion(concepts)
	if err != nil {
		return nil, err
	}
	return &Seed{
		Version:  version,
		Concepts: concepts,
		Edges:    buildEdges(concepts),
		Aliases:  buildAliases(concepts),
	}, nil
}

func normalizeConcept(c *Concept) {
	c.ID = strings.TrimSpace(c.ID)
	c.Name = strings.TrimSpace(c.Name)
	c.Summary = strings.TrimSpace(c.Summary)
	for i := range c.Objectives {
		c.Objectives[i].Text = strings.TrimSpace(c.Objectives[i].Text)
		if c.Objectives[i].Depth == 0 {
			c.Objectives[i].Depth = c.Depth
		}
	}
}

func buildEdges(concepts []Concept) []Edge {
	seen := make(map[Edge]struct{})
	var edges []Edge
	add := func(e Edge) {
		if _, ok := seen[e]; ok {
			return
		}
		seen[e] = struct{}{}
		edges = append(edges, e)
	}
	for _, c := range concepts {
		for _, p := range c.Prerequisites {
			add(Edge{From: p, To: c.ID, Kind: EdgePrerequisite})
		}
		for _, r := range c.Related {
			// related is symmetric: store one canonical direction.
			from, to := c.ID, r
			if to < from {
				from, to = to, from
			}
			add(Edge{From: from, To: to, Kind: EdgeRelated})
		}
		for _, parent := range c.PartOf {
			add(Edge{From: c.ID, To: parent, Kind: EdgePartOf})
		}
	}
	return edges
}

func buildAliases(concepts []Concept) []Alias {
	seen := make(map[string]struct{})
	var out []Alias
	for _, c := range concepts {
		for _, raw := range append([]string{c.Name}, c.Aliases...) {
			key := NormalizeAlias(raw)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, Alias{Alias: key, Display: strings.TrimSpace(raw), ConceptID: c.ID})
		}
	}
	return out
}

// computeVersion hashes the canonical JSON of the sorted, normalized concepts,
// so formatting/comment/file-order changes do not create a new version.
func computeVersion(sorted []Concept) (string, error) {
	payload, err := json.Marshal(sorted)
	if err != nil {
		return "", fmt.Errorf("hash curriculum: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}
