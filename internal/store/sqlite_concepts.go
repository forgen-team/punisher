package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
)

// ConceptFilter narrows ListConcepts. Empty fields mean "any";
// Status defaults to active at the handler level.
type ConceptFilter struct {
	Area   string
	Status string
}

const conceptColumns = `
	c.id, c.name, c.area, c.depth, c.summary, c.practical_cases, c.doc_links,
	c.status, c.origin, COALESCE(c.curriculum_version, ''), c.updated_at,
	(SELECT COUNT(*) FROM objectives o WHERE o.concept_id = c.id AND o.status = 'active'),
	COALESCE(s.state, 'not_started')`

func scanConcept(row rowScanner) (models.Concept, error) {
	var c models.Concept
	err := row.Scan(&c.ID, &c.Name, &c.Area, &c.Depth, &c.Summary, rawJSON(&c.PracticalCases), rawJSON(&c.DocLinks),
		&c.Status, &c.Origin, &c.CurriculumVersion, ts(&c.UpdatedAt), &c.ObjectiveCount, &c.UserState)
	return c, err
}

// ListConcepts returns concepts with the user's state, ordered by area, depth, id.
func (s *SQLiteStore) ListConcepts(ctx context.Context, userID string, f ConceptFilter) ([]models.Concept, error) {
	concepts, err := collect(s.c().query(ctx, `
		SELECT `+conceptColumns+`
		FROM concepts c
		LEFT JOIN user_concept_states s ON s.concept_id = c.id AND s.user_id = $1
		WHERE ($2 = '' OR c.area = $2) AND ($3 = '' OR c.status = $3)
		ORDER BY c.area, c.depth, c.id`, userID, f.Area, f.Status), scanConcept)
	if err != nil {
		return nil, fmt.Errorf("list concepts: %w", err)
	}
	return concepts, nil
}

// GetConceptDetail returns one concept with aliases, objectives, neighbours and lock state.
func (s *SQLiteStore) GetConceptDetail(ctx context.Context, userID, conceptID string) (*models.ConceptDetail, error) {
	c, err := scanConcept(s.c().row(ctx, `
		SELECT `+conceptColumns+`
		FROM concepts c
		LEFT JOIN user_concept_states s ON s.concept_id = c.id AND s.user_id = $1
		WHERE c.id = $2`, userID, conceptID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get concept: %w", err)
	}

	detail := &models.ConceptDetail{Concept: c}
	if detail.Aliases, err = collect(s.c().query(ctx,
		`SELECT display FROM concept_aliases WHERE concept_id = $1 ORDER BY display`, conceptID), scanString); err != nil {
		return nil, fmt.Errorf("list aliases: %w", err)
	}
	if detail.Objectives, err = s.conceptObjectives(ctx, userID, conceptID); err != nil {
		return nil, err
	}
	if err := s.fillNeighbours(ctx, userID, detail); err != nil {
		return nil, err
	}
	if !learning.IsDone(c.UserState) {
		for _, p := range detail.Prerequisites {
			if !learning.IsDone(p.UserState) {
				detail.Locked = true
				break
			}
		}
	}
	return detail, nil
}

func (s *SQLiteStore) conceptObjectives(ctx context.Context, userID, conceptID string) ([]models.Objective, error) {
	objectives, err := collect(s.c().query(ctx, `
		SELECT o.id, o.text, o.depth, o.ordinal, COALESCE(us.state, 'not_started')
		FROM objectives o
		LEFT JOIN user_objective_states us
		       ON us.objective_id = o.id AND us.depth = o.depth AND us.user_id = $1
		WHERE o.concept_id = $2 AND o.status = 'active'
		ORDER BY o.ordinal`, userID, conceptID), func(row rowScanner) (models.Objective, error) {
		var o models.Objective
		err := row.Scan(&o.ID, &o.Text, &o.Depth, &o.Ordinal, &o.UserState)
		return o, err
	})
	if err != nil {
		return nil, fmt.Errorf("list objectives: %w", err)
	}
	return objectives, nil
}

// fillNeighbours loads prerequisite / dependent / related neighbours that are not retired.
func (s *SQLiteStore) fillNeighbours(ctx context.Context, userID string, d *models.ConceptDetail) error {
	type neighbour struct {
		direction string
		link      models.ConceptLink
	}
	rows, err := collect(s.c().query(ctx, `
		SELECT e.kind,
		       CASE WHEN e.to_id = $2 THEN 'in' ELSE 'out' END AS direction,
		       n.id, n.name, n.area, COALESCE(s.state, 'not_started')
		FROM concept_edges e
		JOIN concepts n ON n.id = CASE WHEN e.to_id = $2 THEN e.from_id ELSE e.to_id END
		LEFT JOIN user_concept_states s ON s.concept_id = n.id AND s.user_id = $1
		WHERE (e.from_id = $2 OR e.to_id = $2) AND n.status <> 'retired'
		ORDER BY n.area, n.depth, n.id`, userID, d.ID), func(row rowScanner) (neighbour, error) {
		var n neighbour
		err := row.Scan(&n.link.Kind, &n.direction, &n.link.ID, &n.link.Name, &n.link.Area, &n.link.UserState)
		return n, err
	})
	if err != nil {
		return fmt.Errorf("list neighbours: %w", err)
	}

	d.Prerequisites, d.Dependents, d.Related = []models.ConceptLink{}, []models.ConceptLink{}, []models.ConceptLink{}
	for _, n := range rows {
		switch {
		case n.link.Kind == "prerequisite" && n.direction == "in":
			d.Prerequisites = append(d.Prerequisites, n.link)
		case n.link.Kind == "prerequisite":
			d.Dependents = append(d.Dependents, n.link)
		default:
			d.Related = append(d.Related, n.link)
		}
	}
	return nil
}

func scanEdge(row rowScanner) (models.Edge, error) {
	var e models.Edge
	err := row.Scan(&e.From, &e.To, &e.Kind)
	return e, err
}

// LoadGraph loads active concepts, edges among them and the user's states.
func (s *SQLiteStore) LoadGraph(ctx context.Context, userID string) (*models.GraphData, error) {
	concepts, err := s.ListConcepts(ctx, userID, ConceptFilter{Status: "active"})
	if err != nil {
		return nil, err
	}
	version, err := s.LatestCurriculumVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("curriculum version: %w", err)
	}
	edges, err := collect(s.c().query(ctx, `
		SELECT e.from_id, e.to_id, e.kind
		FROM concept_edges e
		JOIN concepts a ON a.id = e.from_id AND a.status = 'active'
		JOIN concepts b ON b.id = e.to_id AND b.status = 'active'
		ORDER BY e.kind, e.from_id, e.to_id`), scanEdge)
	if err != nil {
		return nil, fmt.Errorf("list edges: %w", err)
	}

	states := make(map[string]string, len(concepts))
	for _, c := range concepts {
		states[c.ID] = c.UserState
	}
	return &models.GraphData{CurriculumVersion: version, Concepts: concepts, Edges: edges, States: states}, nil
}

// ResolveConcepts finds non-retired concepts whose normalized alias/name or id
// exactly equals the normalized query. No fuzzy or embedding matching yet.
func (s *SQLiteStore) ResolveConcepts(ctx context.Context, normalizedQuery string) ([]string, error) {
	ids, err := collect(s.c().query(ctx, `
		SELECT DISTINCT c.id
		FROM concepts c
		LEFT JOIN concept_aliases a ON a.concept_id = c.id
		WHERE c.status <> 'retired' AND (a.alias = $1 OR c.id = $1)
		ORDER BY c.id`, normalizedQuery), scanString)
	if err != nil {
		return nil, fmt.Errorf("resolve concepts: %w", err)
	}
	return ids, nil
}
