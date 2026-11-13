package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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
	// Review: the review queue counts now (learning.TodayInput.Review).
	Review learning.ReviewCounts
	// Lapsed: concepts with an objective lapsed learning.RecheckLapses+ times (learning.TodayInput.Lapsed).
	Lapsed map[string]learning.LapsedConcept
}

// GoalTargets is an active goal's targets.
type GoalTargets struct {
	ID      string
	Title   string
	Targets []string
}

// LoadToday loads concepts with pending-objective flags, prerequisite edges,
// in-progress lectures, resolved study requests, active goals and the review queue counts.
func (s *SQLiteStore) LoadToday(ctx context.Context, userID string, now time.Time) (*TodayData, error) {
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
		SELECT l.id, l.concept_id, l.title, p.anchor, p.percent, l.estimated_minutes, `+parentLectureSQL("l.id")+`,
		       CASE WHEN l.minutes_estimated THEN NULL ELSE l.content END
		FROM user_lecture_progress p
		JOIN lectures l ON l.id = p.lecture_id
		WHERE p.user_id = $1 AND p.completed_at IS NULL AND p.dismissed_at IS NULL AND p.percent < 100
		ORDER BY p.last_seen_at DESC LIMIT 200`, userID), func(row rowScanner) (learning.ResumeItem, error) {
		var r learning.ResumeItem
		var legacy json.RawMessage // 기존 강의(섹션 분 합계)만 본문을 읽어 조회 때 계산한다.
		err := row.Scan(&r.LectureID, &r.ConceptID, &r.Title, &r.Anchor, &r.Percent, &r.EstimatedMinutes, &r.ParentLectureID, rawJSON(&legacy))
		if err == nil && legacy != nil {
			r.EstimatedMinutes = contentMinutes(legacy, r.EstimatedMinutes)
		}
		return r, err
	}); err != nil {
		return nil, fmt.Errorf("today resume: %w", err)
	}
	newer, err := s.regeneratedVersions(ctx, userID)
	if err != nil {
		return nil, err
	}
	d.Resume = learning.ReplaceRegenerated(d.Resume, newer)

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
	if d.Review, err = s.ReviewCounts(ctx, userID, now); err != nil {
		return nil, err
	}
	if d.Lapsed, err = s.lapsedConcepts(ctx, userID, now); err != nil {
		return nil, err
	}
	return d, nil
}

// lapsedConcepts: concepts with an active objective whose lapses (reset on
// recovery) reached learning.RecheckLapses — due when one of them is in the
// review queue now, with the lecture of the concept the user last opened.
// SQLite: bool_or(x) → MAX(x) (0/1).
func (s *SQLiteStore) lapsedConcepts(ctx context.Context, userID string, now time.Time) (map[string]learning.LapsedConcept, error) {
	type lapsed struct {
		id string
		c  learning.LapsedConcept
	}
	list, err := collect(s.c().query(ctx, `
		SELECT m.concept_id, MAX(m.next_review_at <= $2 AND NOT m.queued AND m.grad_step < 2),
			COALESCE((SELECT l.id FROM user_lecture_progress p JOIN lectures l ON l.id = p.lecture_id
				WHERE p.user_id = $1 AND l.concept_id = m.concept_id ORDER BY p.last_seen_at DESC LIMIT 1), '')
		FROM user_objective_mastery m`+reviewScope+`
		WHERE m.user_id = $1 AND m.lapses >= $3
		GROUP BY m.concept_id`, userID, now, learning.RecheckLapses), func(row rowScanner) (lapsed, error) {
		var r lapsed
		var due sql.NullBool
		err := row.Scan(&r.id, &due, &r.c.LectureID)
		r.c.Due = due.Bool
		return r, err
	})
	if err != nil {
		return nil, fmt.Errorf("lapsed concepts: %w", err)
	}
	out := map[string]learning.LapsedConcept{}
	for _, r := range list {
		out[r.id] = r.c
	}
	return out, nil
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

// regeneratedVersions maps a lecture remade with '다시 만들기' to its newest
// published version, for 이어보기 (learning.ReplaceRegenerated).
func (s *SQLiteStore) regeneratedVersions(ctx context.Context, userID string) (map[string]learning.NewerVersion, error) {
	type row struct {
		parent string
		n      learning.NewerVersion
	}
	rows, err := collect(s.c().query(ctx, `
		SELECT k.parent_lecture_id, c.id, c.concept_id, c.title, c.estimated_minutes,
		       CASE WHEN c.minutes_estimated THEN NULL ELSE c.content END,
		       EXISTS (SELECT 1 FROM user_lecture_progress q WHERE q.user_id = $1 AND q.lecture_id = c.id)
		FROM lecture_links k
		JOIN lectures c ON c.id = k.child_lecture_id
		WHERE k.user_id = $1 AND k.kind = $2 AND c.status = 'published'
		ORDER BY k.created_at`, userID, LinkRegeneratedKind), func(rs rowScanner) (row, error) {
		var r row
		var legacy json.RawMessage
		err := rs.Scan(&r.parent, &r.n.Item.LectureID, &r.n.Item.ConceptID, &r.n.Item.Title, &r.n.Item.EstimatedMinutes, rawJSON(&legacy), &r.n.Started)
		if err == nil && legacy != nil {
			r.n.Item.EstimatedMinutes = contentMinutes(legacy, r.n.Item.EstimatedMinutes)
		}
		return r, err
	})
	if err != nil {
		return nil, fmt.Errorf("today regenerated versions: %w", err)
	}
	out := make(map[string]learning.NewerVersion, len(rows))
	for _, r := range rows {
		out[r.parent] = r.n // ordered by link time: the newest version wins
	}
	return out, nil
}
