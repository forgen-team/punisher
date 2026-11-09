// Package chapter validates and normalizes a generated "next chapter" (a
// user's own group of 5-12 concepts with objectives and prerequisite edges).
// It is pure: the store applies a validated Draft, services call the LLM.
package chapter

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	MinConcepts      = 5
	MaxConcepts      = 12
	MaxObjectives    = 6
	MaxPrerequisites = 4
	maxTitleRunes    = 60
	maxNameRunes     = 40
	maxTextRunes     = 200
	maxSummaryRunes  = 300
)

var (
	keyRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// termLinkRe: lecture-body link syntax the model sometimes copies into chapter text.
	termLinkRe = regexp.MustCompile(`\[([^\]\n]+)\]\(term:[^)\s]*\)`)
)

// plainText trims and drops lecture link markup ([용어](term:id) → 용어).
func plainText(s string) string {
	return strings.TrimSpace(termLinkRe.ReplaceAllString(s, "$1"))
}

// Objective is a learning objective of a chapter concept.
type Objective struct {
	Text  string `json:"text"`
	Depth int    `json:"depth"`
}

// Concept is one chapter entry: a new concept (Key), a reused existing concept
// (ExistingID) or one of the user's proposed concepts to promote (ProposedID).
type Concept struct {
	Key           string      `json:"key"`
	ExistingID    string      `json:"existing_id,omitempty"`
	ProposedID    string      `json:"proposed_id,omitempty"`
	Name          string      `json:"name"`
	Area          string      `json:"area"`
	Depth         int         `json:"depth"`
	Summary       string      `json:"summary"`
	Objectives    []Objective `json:"objectives"`
	Prerequisites []string    `json:"prerequisites"`
	// ID is assigned by Validate: the final concept id (new: u<hash>.<key>).
	ID string `json:"id,omitempty"`
	// Action is assigned by Validate: create | reuse | promote.
	Action string `json:"action,omitempty"`
}

// Draft is the LLM output (after Validate: normalized, topologically ordered).
type Draft struct {
	Title    string    `json:"title"`
	Summary  string    `json:"summary"`
	Concepts []Concept `json:"concepts"`
}

// Actions.
const (
	ActionCreate  = "create"
	ActionReuse   = "reuse"
	ActionPromote = "promote"
)

// Context is what the validator needs about the user's map.
type Context struct {
	UserID string
	// Visible: concept id → name of every concept on the user's map (seed + own chapters).
	Visible map[string]string
	// OwnProposed: the user's proposed concept ids (may be promoted).
	OwnProposed map[string]string
	// Areas: allowed area ids.
	Areas []string
}

// OwnerPrefix is the id namespace of a user's chapter concepts.
func OwnerPrefix(userID string) string {
	sum := sha256.Sum256([]byte(userID))
	return "u" + hex.EncodeToString(sum[:])[:8]
}

// NormalizeName folds case and whitespace for duplicate-name checks.
func NormalizeName(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// Validate checks and normalizes d. It returns the normalized draft (ids and
// actions assigned, concepts in prerequisite order) or the problems found.
// Rules: 5-12 concepts; keys are lowercase-hyphen slugs, unique; existing ids
// must be on the user's map and are reused rather than recreated (also when a
// new concept's name equals a visible concept's name); proposed ids must be
// the user's own; prerequisites reference visible ids or keys of this draft,
// at most 4, and the prerequisite graph has no cycle.
func Validate(d Draft, c Context) (Draft, []string) {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	d.Title = strings.TrimSpace(d.Title)
	if d.Title == "" || utf8.RuneCountInString(d.Title) > maxTitleRunes {
		add("title is required (max %d chars)", maxTitleRunes)
	}
	d.Summary = clip(plainText(d.Summary), maxSummaryRunes)
	if n := len(d.Concepts); n < MinConcepts || n > MaxConcepts {
		add("chapter needs %d-%d concepts, got %d", MinConcepts, MaxConcepts, n)
	}

	areas := map[string]bool{}
	for _, a := range c.Areas {
		areas[a] = true
	}
	byName := map[string]string{}
	for id, name := range c.Visible {
		byName[NormalizeName(name)] = id
	}
	prefix := OwnerPrefix(c.UserID)
	keyToID := map[string]string{}
	seenIDs := map[string]bool{}
	for i := range d.Concepts {
		x := &d.Concepts[i]
		x.Key = strings.TrimSpace(strings.ToLower(x.Key))
		x.Name = strings.TrimSpace(x.Name)
		where := fmt.Sprintf("concepts[%d]", i)
		switch {
		case x.ExistingID != "":
			if _, ok := c.Visible[x.ExistingID]; !ok {
				add("%s: existing_id %q is not on the map", where, x.ExistingID)
				continue
			}
			x.ID, x.Action, x.Name = x.ExistingID, ActionReuse, c.Visible[x.ExistingID]
		case x.ProposedID != "":
			name, ok := c.OwnProposed[x.ProposedID]
			if !ok {
				add("%s: proposed_id %q is not one of the user's proposed concepts", where, x.ProposedID)
				continue
			}
			x.ID, x.Action = x.ProposedID, ActionPromote
			if x.Name == "" {
				x.Name = name
			}
		default:
			if id, dup := byName[NormalizeName(x.Name)]; dup && x.Name != "" {
				x.ID, x.Action, x.Name = id, ActionReuse, c.Visible[id] // reuse rather than duplicate
				break
			}
			if !keyRe.MatchString(x.Key) {
				add("%s: key %q must be a lowercase-hyphen slug", where, x.Key)
				continue
			}
			x.ID, x.Action = prefix+"."+x.Key, ActionCreate
		}
		if x.Key == "" {
			x.Key = x.ID
		}
		if seenIDs[x.ID] {
			if x.Action == ActionReuse {
				x.ID = "" // the same existing concept listed twice: keep the first
				continue
			}
			add("%s: duplicate concept %q", where, x.ID)
			continue
		}
		seenIDs[x.ID] = true
		keyToID[x.Key] = x.ID
		keyToID[x.ID] = x.ID // a validated draft references chapter concepts by id
		if x.Action == ActionReuse {
			continue
		}
		validateNew(where, x, areas, add)
	}

	for i := range d.Concepts {
		x := &d.Concepts[i]
		if x.Action == ActionReuse || x.ID == "" {
			continue
		}
		if len(x.Prerequisites) > MaxPrerequisites {
			add("concepts[%d]: at most %d prerequisites", i, MaxPrerequisites)
		}
		resolved := []string{}
		for _, p := range x.Prerequisites {
			p = strings.TrimSpace(p)
			id, ok := keyToID[strings.ToLower(p)]
			if !ok {
				if _, visible := c.Visible[p]; visible {
					id, ok = p, true
				}
			}
			if !ok {
				add("concepts[%d]: prerequisite %q is neither on the map nor in this chapter", i, p)
				continue
			}
			if id == x.ID {
				add("concepts[%d]: a concept cannot be its own prerequisite", i)
				continue
			}
			resolved = append(resolved, id)
		}
		x.Prerequisites = dedupe(resolved)
	}
	if len(problems) > 0 {
		return d, problems
	}
	kept := d.Concepts[:0]
	for _, x := range d.Concepts {
		if x.ID != "" {
			kept = append(kept, x)
		}
	}
	d.Concepts = kept
	if n := len(d.Concepts); n < MinConcepts {
		return d, []string{fmt.Sprintf("chapter needs %d-%d distinct concepts, got %d", MinConcepts, MaxConcepts, n)}
	}
	ordered, err := topoOrder(d.Concepts)
	if err != nil {
		return d, []string{err.Error()}
	}
	d.Concepts = ordered
	return d, nil
}

func validateNew(where string, x *Concept, areas map[string]bool, add func(string, ...any)) {
	if x.Name == "" || utf8.RuneCountInString(x.Name) > maxNameRunes {
		add("%s: name is required (max %d chars)", where, maxNameRunes)
	}
	if !areas[x.Area] {
		add("%s: area %q must be one of the existing areas", where, x.Area)
	}
	if x.Depth < 1 || x.Depth > 3 {
		add("%s: depth must be 1..3", where)
	}
	x.Summary = clip(plainText(x.Summary), maxSummaryRunes)
	if n := len(x.Objectives); n < 1 || n > MaxObjectives {
		add("%s: needs 1-%d objectives", where, MaxObjectives)
	}
	for j := range x.Objectives {
		o := &x.Objectives[j]
		o.Text = plainText(o.Text)
		if o.Text == "" || utf8.RuneCountInString(o.Text) > maxTextRunes {
			add("%s.objectives[%d]: text is required (max %d chars)", where, j, maxTextRunes)
		}
		if o.Depth < 1 || o.Depth > 3 {
			o.Depth = x.Depth
		}
	}
}

// topoOrder orders concepts so prerequisites inside the chapter come first,
// keeping the draft order otherwise; it fails on a cycle.
func topoOrder(cs []Concept) ([]Concept, error) {
	index := map[string]int{}
	for i, c := range cs {
		index[c.ID] = i
	}
	const (
		unvisited = iota
		visiting
		done
	)
	state := make([]int, len(cs))
	out := make([]Concept, 0, len(cs))
	var visit func(i int, path []string) error
	visit = func(i int, path []string) error {
		switch state[i] {
		case done:
			return nil
		case visiting:
			return fmt.Errorf("prerequisite cycle: %s", strings.Join(append(path, cs[i].ID), " → "))
		}
		state[i] = visiting
		for _, p := range cs[i].Prerequisites {
			if j, inChapter := index[p]; inChapter {
				if err := visit(j, append(path, cs[i].ID)); err != nil {
					return err
				}
			}
		}
		state[i] = done
		out = append(out, cs[i])
		return nil
	}
	for i := range cs {
		if err := visit(i, nil); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func dedupe(ids []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func clip(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
