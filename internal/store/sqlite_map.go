package store

import (
	"context"
	"fmt"

	"github.com/forgen-team/punisher/internal/models"
)

// LoadMapExtras adds what only the knowledge map needs to data: the caller's
// pending study requests, the proposed (dynamic) concepts they resolved to —
// with their prerequisite edges, so a request on a new concept sits next to its
// closest existing concepts — and the concepts whose lectures were all dismissed.
func (s *SQLiteStore) LoadMapExtras(ctx context.Context, userID string, data *models.GraphData) error {
	var err error
	data.Requests, err = collect(s.c().query(ctx, `
		SELECT id, query, source, status, resolved_concept_ids FROM study_requests
		WHERE user_id = $1 AND status IN ('unresolved', 'resolved', 'in_progress')
		ORDER BY created_at DESC LIMIT 100`, userID), func(row rowScanner) (models.MapRequest, error) {
		var r models.MapRequest
		err := row.Scan(&r.ID, &r.Query, &r.Source, &r.Status, strs(&r.ConceptIDs))
		return r, err
	})
	if err != nil {
		return fmt.Errorf("map requests: %w", err)
	}

	data.Inquiries, err = collect(s.c().query(ctx, `
		SELECT id, question, concept_ids, proposed_concept_ids FROM inquiries
		WHERE user_id = $1 AND answered_at IS NOT NULL
		ORDER BY created_at DESC LIMIT 200`, userID), func(row rowScanner) (models.MapInquiry, error) {
		var q models.MapInquiry
		var related, proposed []string
		err := row.Scan(&q.ID, &q.Question, strs(&related), strs(&proposed))
		q.ConceptIDs = append(related, proposed...)
		return q, err
	})
	if err != nil {
		return fmt.Errorf("map inquiries: %w", err)
	}

	loaded := make(map[string]bool, len(data.Concepts))
	for _, c := range data.Concepts {
		loaded[c.ID] = true
	}
	var missing []string
	for _, r := range data.Requests {
		for _, id := range r.ConceptIDs {
			if !loaded[id] {
				loaded[id] = true
				missing = append(missing, id)
			}
		}
	}
	// 질문 해설이 지도에 새로 제안한 개념도 노드로 보인다.
	for _, q := range data.Inquiries {
		for _, id := range q.ConceptIDs {
			if !loaded[id] {
				loaded[id] = true
				missing = append(missing, id)
			}
		}
	}
	if len(missing) > 0 {
		if err := s.addProposedConcepts(ctx, userID, missing, data); err != nil {
			return err
		}
	}

	data.Dismissed, err = s.DismissedConceptIDs(ctx, userID)
	return err
}

// addProposedConcepts appends proposed concepts among ids and their edges to
// already loaded concepts.
func (s *SQLiteStore) addProposedConcepts(ctx context.Context, userID string, ids []string, data *models.GraphData) error {
	proposed, err := collect(s.c().query(ctx, `
		SELECT `+conceptColumns+`
		FROM concepts c
		LEFT JOIN user_concept_states s ON s.concept_id = c.id AND s.user_id = $1
		WHERE c.id IN (SELECT value FROM json_each($2)) AND c.status = 'proposed' AND `+visibleTo("c", "$1")+`
		ORDER BY c.area, c.depth, c.id`, userID, ids), scanConcept)
	if err != nil {
		return fmt.Errorf("proposed concepts: %w", err)
	}
	present := make(map[string]bool, len(data.Concepts))
	for _, c := range data.Concepts {
		present[c.ID] = true
	}
	var added []string
	for _, c := range proposed {
		data.Concepts = append(data.Concepts, c)
		if data.States == nil {
			data.States = map[string]string{}
		}
		data.States[c.ID] = c.UserState
		present[c.ID] = true
		added = append(added, c.ID)
	}
	if len(added) == 0 {
		return nil
	}

	edges, err := collect(s.c().query(ctx, `
		SELECT from_id, to_id, kind FROM concept_edges
		WHERE from_id IN (SELECT value FROM json_each($1)) OR to_id IN (SELECT value FROM json_each($1))`, added), scanEdge)
	if err != nil {
		return fmt.Errorf("proposed edges: %w", err)
	}
	for _, e := range edges {
		if present[e.From] && present[e.To] {
			data.Edges = append(data.Edges, e)
		}
	}
	return nil
}
