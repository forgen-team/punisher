package store

import (
	"context"
	"fmt"

	"github.com/wooo-jin/punisher/internal/curriculum"
	"github.com/wooo-jin/punisher/internal/learning"
)

// TodayData is the raw input for learning.BuildToday plus active goals.
type TodayData struct {
	Concepts map[string]learning.TodayConcept
	Edges    []learning.PrereqEdge
	Resume   []learning.ResumeItem
	Requests []learning.RequestInput
	Goals    []GoalTargets
}

// GoalTargets is an active goal's targets.
type GoalTargets struct {
	ID      string
	Title   string
	Targets []string
}

// LoadToday loads concepts with pending-objective flags, prerequisite edges,
// in-progress lectures, resolved study requests and active goals.
func (s *SQLiteStore) LoadToday(ctx context.Context, userID string) (*TodayData, error) {
	d := &TodayData{Concepts: map[string]learning.TodayConcept{}}

	concepts, err := collect(s.c().query(ctx, `
		SELECT c.id, c.name, c.area, c.depth, c.summary, COALESCE(json_extract(c.practical_cases, '$[0]'), ''),
		       COALESCE(st.state, 'not_started'), c.status = 'proposed', c.owner_user_id IS NOT NULL,
		       (
		         SELECT COUNT(DISTINCT o.depth) FROM objectives o
		         WHERE o.concept_id = c.id AND o.status = 'active'
		           AND NOT EXISTS (
		             SELECT 1 FROM user_objective_states us
		             WHERE us.user_id = $1 AND us.objective_id = o.id AND us.depth = o.depth
		               AND us.state IN ('completed', 'known')))
		FROM concepts c
		LEFT JOIN user_concept_states st ON st.concept_id = c.id AND st.user_id = $1
		WHERE c.status IN ('active', 'proposed') AND `+visibleTo("c", "$1"), userID), func(row rowScanner) (learning.TodayConcept, error) {
		var c learning.TodayConcept
		err := row.Scan(&c.ID, &c.Name, &c.Area, &c.Depth, &c.Summary, &c.PracticalCase, &c.State, &c.Proposed,
			&c.Private, &c.PendingDepths)
		c.HasPending = c.PendingDepths > 0
		return c, err
	})
	if err != nil {
		return nil, fmt.Errorf("today concepts: %w", err)
	}
	for _, c := range concepts {
		d.Concepts[c.ID] = c
	}

	if d.Edges, err = s.PrerequisiteEdges(ctx, userID); err != nil {
		return nil, err
	}

	if d.Resume, err = collect(s.c().query(ctx, `
		SELECT l.id, l.concept_id, l.title, p.anchor, p.percent, l.estimated_minutes, `+parentLectureSQL("l.id")+`
		FROM user_lecture_progress p
		JOIN lectures l ON l.id = p.lecture_id
		WHERE p.user_id = $1 AND p.completed_at IS NULL AND p.dismissed_at IS NULL AND p.percent < 100
		ORDER BY p.last_seen_at DESC LIMIT 5`, userID), func(row rowScanner) (learning.ResumeItem, error) {
		var r learning.ResumeItem
		err := row.Scan(&r.LectureID, &r.ConceptID, &r.Title, &r.Anchor, &r.Percent, &r.EstimatedMinutes, &r.ParentLectureID)
		return r, err
	}); err != nil {
		return nil, fmt.Errorf("today resume: %w", err)
	}

	if d.Requests, err = collect(s.c().query(ctx, `
		SELECT id, query, resolved_concept_ids FROM study_requests
		WHERE user_id = $1 AND status IN ('resolved', 'in_progress') AND json_array_length(resolved_concept_ids) > 0
		ORDER BY created_at LIMIT 20`, userID), func(row rowScanner) (learning.RequestInput, error) {
		var r learning.RequestInput
		err := row.Scan(&r.ID, &r.Query, strs(&r.ConceptIDs))
		return r, err
	}); err != nil {
		return nil, fmt.Errorf("today requests: %w", err)
	}

	if d.Goals, err = collect(s.c().query(ctx, `
		SELECT id, title, target_concept_ids FROM user_goals
		WHERE user_id = $1 AND status = 'active' ORDER BY updated_at DESC`, userID), func(row rowScanner) (GoalTargets, error) {
		var g GoalTargets
		err := row.Scan(&g.ID, &g.Title, strs(&g.Targets))
		return g, err
	}); err != nil {
		return nil, fmt.Errorf("today goals: %w", err)
	}
	return d, nil
}

// PrerequisiteEdges returns prerequisite edges between non-retired concepts.
func (s *SQLiteStore) PrerequisiteEdges(ctx context.Context, userID string) ([]learning.PrereqEdge, error) {
	edges, err := collect(s.c().query(ctx, `
		SELECT e.from_id, e.to_id FROM concept_edges e
		JOIN concepts a ON a.id = e.from_id AND a.status <> 'retired' AND `+visibleTo("a", "$2")+`
		JOIN concepts b ON b.id = e.to_id AND b.status <> 'retired' AND `+visibleTo("b", "$2")+`
		WHERE e.kind = $1`, curriculum.EdgePrerequisite, userID), func(row rowScanner) (learning.PrereqEdge, error) {
		var e learning.PrereqEdge
		err := row.Scan(&e.From, &e.To)
		return e, err
	})
	if err != nil {
		return nil, fmt.Errorf("prerequisite edges: %w", err)
	}
	return edges, nil
}

// AliasIndex loads normalized aliases (and ids) of non-retired concepts.
func (s *SQLiteStore) AliasIndex(ctx context.Context, userID string) (learning.AliasIndex, error) {
	type pair struct{ alias, id string }
	pairs, err := collect(s.c().query(ctx, `
		SELECT a.alias, a.concept_id FROM concept_aliases a
		JOIN concepts c ON c.id = a.concept_id AND c.status <> 'retired' AND `+visibleTo("c", "$1")+`
		UNION ALL SELECT c.id, c.id FROM concepts c WHERE c.status <> 'retired' AND `+visibleTo("c", "$1"), userID), func(row rowScanner) (pair, error) {
		var p pair
		err := row.Scan(&p.alias, &p.id)
		return p, err
	})
	if err != nil {
		return nil, fmt.Errorf("alias index: %w", err)
	}
	idx := learning.AliasIndex{}
	for _, p := range pairs {
		idx[p.alias] = p.id
	}
	return idx, nil
}
