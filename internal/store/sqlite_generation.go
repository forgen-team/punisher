package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/wooo-jin/punisher/internal/curriculum"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
)

// ConceptInfo loads a non-retired concept visible to userID with its active objectives.
func (s *SQLiteStore) ConceptInfo(ctx context.Context, userID, conceptID string) (models.ConceptInfo, error) {
	var c models.ConceptInfo
	var cases, links json.RawMessage
	err := s.c().row(ctx, `
		SELECT id, name, area, depth, summary, status, practical_cases, doc_links
		FROM concepts c WHERE c.id = $1 AND c.status <> 'retired' AND `+visibleTo("c", "$2"), conceptID, userID).
		Scan(&c.ID, &c.Name, &c.Area, &c.Depth, &c.Summary, &c.Status, rawJSON(&cases), rawJSON(&links))
	if errors.Is(err, sql.ErrNoRows) {
		return c, ErrNotFound
	}
	if err != nil {
		return c, fmt.Errorf("load concept info: %w", err)
	}
	if err := json.Unmarshal(cases, &c.PracticalCases); err != nil {
		return c, fmt.Errorf("decode practical_cases: %w", err)
	}
	if err := json.Unmarshal(links, &c.DocLinks); err != nil {
		return c, fmt.Errorf("decode doc_links: %w", err)
	}
	c.Objectives, err = collect(s.c().query(ctx, `
		SELECT id, text, depth, ordinal FROM objectives
		WHERE concept_id = $1 AND status = 'active' ORDER BY ordinal`, conceptID),
		func(row rowScanner) (models.ObjectiveInfo, error) {
			var o models.ObjectiveInfo
			err := row.Scan(&o.ID, &o.Text, &o.Depth, &o.Ordinal)
			return o, err
		})
	if err != nil {
		return c, fmt.Errorf("load objectives: %w", err)
	}
	return c, nil
}

// DoneSet returns the user's completed/known (objective, depth) pairs for a concept.
func (s *SQLiteStore) DoneSet(ctx context.Context, userID, conceptID string) (learning.DoneSet, error) {
	type pair struct {
		id    string
		depth int
	}
	pairs, err := collect(s.c().query(ctx, `
		SELECT objective_id, depth FROM user_objective_states
		WHERE user_id = $1 AND concept_id = $2 AND state IN ('completed', 'known')`, userID, conceptID),
		func(row rowScanner) (pair, error) {
			var p pair
			err := row.Scan(&p.id, &p.depth)
			return p, err
		})
	if err != nil {
		return nil, fmt.Errorf("load done set: %w", err)
	}
	done := learning.DoneSet{}
	for _, p := range pairs {
		done[learning.DoneKey(p.id, p.depth)] = true
	}
	return done, nil
}

// ConceptRef is a minimal concept reference.
type ConceptRef struct {
	ID      string
	Name    string
	Area    string
	Depth   int
	Summary string
	State   string
}

func scanConceptRef(row rowScanner) (ConceptRef, error) {
	var c ConceptRef
	err := row.Scan(&c.ID, &c.Name, &c.Area, &c.Depth, &c.Summary, &c.State)
	return c, err
}

// ConceptCatalog lists active and proposed concepts with the user's state.
func (s *SQLiteStore) ConceptCatalog(ctx context.Context, userID string) ([]ConceptRef, error) {
	refs, err := collect(s.c().query(ctx, `
		SELECT c.id, c.name, c.area, c.depth, c.summary, COALESCE(st.state, 'not_started')
		FROM concepts c
		LEFT JOIN user_concept_states st ON st.concept_id = c.id AND st.user_id = $1
		WHERE c.status IN ('active', 'proposed') AND `+visibleTo("c", "$1")+`
		ORDER BY c.area, c.depth, c.id`, userID), scanConceptRef)
	if err != nil {
		return nil, fmt.Errorf("concept catalog: %w", err)
	}
	return refs, nil
}

// RelatedConcepts returns prerequisite (in) and related/part_of neighbours of a concept.
func (s *SQLiteStore) RelatedConcepts(ctx context.Context, userID, conceptID string) ([]ConceptRef, error) {
	refs, err := collect(s.c().query(ctx, `
		SELECT DISTINCT n.id, n.name, n.area, n.depth, n.summary, COALESCE(st.state, 'not_started')
		FROM concept_edges e
		JOIN concepts n ON n.id = CASE WHEN e.to_id = $2 THEN e.from_id ELSE e.to_id END
		LEFT JOIN user_concept_states st ON st.concept_id = n.id AND st.user_id = $1
		WHERE (e.from_id = $2 OR e.to_id = $2) AND n.status <> 'retired' AND `+visibleTo("n", "$1"), userID, conceptID), scanConceptRef)
	if err != nil {
		return nil, fmt.Errorf("related concepts: %w", err)
	}
	return refs, nil
}

// PrerequisitesOf returns the direct prerequisites of conceptID with the user's state.
func (s *SQLiteStore) PrerequisitesOf(ctx context.Context, userID, conceptID string) ([]ConceptRef, error) {
	refs, err := collect(s.c().query(ctx, `
		SELECT n.id, n.name, n.area, n.depth, n.summary, COALESCE(st.state, 'not_started')
		FROM concept_edges e
		JOIN concepts n ON n.id = e.from_id AND n.status <> 'retired' AND `+visibleTo("n", "$1")+`
		LEFT JOIN user_concept_states st ON st.concept_id = n.id AND st.user_id = $1
		WHERE e.to_id = $2 AND e.kind = 'prerequisite'
		ORDER BY n.depth, n.id`, userID, conceptID), scanConceptRef)
	if err != nil {
		return nil, fmt.Errorf("prerequisites: %w", err)
	}
	return refs, nil
}

// KnownConceptIDs returns the ids of every non-retired concept visible to userID.
func (s *SQLiteStore) KnownConceptIDs(ctx context.Context, userID string) (map[string]bool, error) {
	ids, err := collect(s.c().query(ctx, `SELECT c.id FROM concepts c WHERE c.status <> 'retired' AND `+visibleTo("c", "$1"), userID), scanString)
	if err != nil {
		return nil, fmt.Errorf("known concepts: %w", err)
	}
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// ProposeConcept adds a dynamic, proposed concept with aliases, objectives and
// prerequisite edges to existing concepts. Existing ids/aliases are left
// untouched (idempotent). Returns the concept id.
func (s *SQLiteStore) ProposeConcept(ctx context.Context, userID string, p models.ProposedConcept) (string, error) {
	err := s.tx(ctx, func(c conn) error {
		if _, err := c.exec(ctx, `
			INSERT INTO concepts (id, name, area, depth, summary, status, origin, proposed_by)
			VALUES ($1, $2, $3, $4, $5, 'proposed', 'dynamic', $6)
			ON CONFLICT (id) DO NOTHING`, p.ID, p.Name, p.Area, p.Depth, p.Summary, userID); err != nil {
			return fmt.Errorf("insert proposed concept: %w", err)
		}
		for _, alias := range append([]string{p.Name}, p.Aliases...) {
			norm := curriculum.NormalizeAlias(alias)
			if norm == "" {
				continue
			}
			if _, err := c.exec(ctx, `
				INSERT INTO concept_aliases (alias, display, concept_id, origin) VALUES ($1, $2, $3, 'dynamic')
				ON CONFLICT (alias) DO NOTHING`, norm, alias, p.ID); err != nil {
				return fmt.Errorf("insert alias: %w", err)
			}
		}
		for i, text := range p.Objectives {
			if _, err := c.exec(ctx, `
				INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT DO NOTHING`, curriculum.ObjectiveID(p.ID, i+1), p.ID, text, p.Depth, i+1); err != nil {
				return fmt.Errorf("insert objective: %w", err)
			}
		}
		for _, pre := range p.Prerequisites {
			if pre == p.ID {
				continue
			}
			if _, err := c.exec(ctx, `
				INSERT INTO concept_edges (from_id, to_id, kind, origin)
				SELECT $1, $2, 'prerequisite', 'dynamic'
				WHERE EXISTS (SELECT 1 FROM concepts WHERE id = $1 AND status <> 'retired')
				ON CONFLICT DO NOTHING`, pre, p.ID); err != nil {
				return fmt.Errorf("insert edge: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return p.ID, nil
}

// GetGloss returns a cached term gloss ("" when absent).
func (s *SQLiteStore) GetGloss(ctx context.Context, conceptID string) (string, error) {
	var gloss string
	err := s.c().row(ctx, `SELECT gloss FROM term_glosses WHERE concept_id = $1`, conceptID).Scan(&gloss)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return gloss, err
}

// SaveGloss caches a gloss (first writer wins).
func (s *SQLiteStore) SaveGloss(ctx context.Context, conceptID, gloss, model string) error {
	_, err := s.c().exec(ctx, `
		INSERT INTO term_glosses (concept_id, gloss, model) VALUES ($1, $2, $3)
		ON CONFLICT (concept_id) DO NOTHING`, conceptID, gloss, model)
	return err
}
