package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/learning"
)

// SynthesisCandidates are the user's objectives with ladder evidence (box ≥ 1,
// not stale) on active objectives of visible concepts (v3 §5).
func (s *SQLiteStore) SynthesisCandidates(ctx context.Context, userID string) ([]learning.SynthesisCandidate, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT m.objective_id, o.concept_id, c.area, m.last_credit_at, m.integrative_ok
		FROM user_objective_mastery m
		JOIN objectives o ON o.id = m.objective_id AND o.status = 'active'
		JOIN concepts c ON c.id = o.concept_id AND c.status = 'active'
		WHERE m.user_id = $1 AND m.box >= 1 AND NOT m.stale AND `+visibleTo("c", "$1")+`
		ORDER BY m.objective_id`, userID), func(r rowScanner) (learning.SynthesisCandidate, error) {
		var c learning.SynthesisCandidate
		err := r.Scan(&c.ObjectiveID, &c.ConceptID, &c.Area, nts(&c.LastCreditAt), &c.IntegrativeOK)
		return c, err
	})
	if err != nil {
		return nil, fmt.Errorf("synthesis candidates: %w", err)
	}
	return list, nil
}

// ConceptNeighbors maps each concept to the concepts linked to it
// (prerequisite / related, either direction).
func (s *SQLiteStore) ConceptNeighbors(ctx context.Context, conceptIDs []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(conceptIDs) == 0 {
		return out, nil
	}
	type edge struct{ a, b string }
	edges, err := collect(s.c().query(ctx, `SELECT from_id, to_id FROM concept_edges
		WHERE kind IN ('prerequisite', 'related')
		AND (from_id IN (SELECT value FROM json_each($1)) OR to_id IN (SELECT value FROM json_each($1)))`, conceptIDs),
		func(r rowScanner) (edge, error) {
			var e edge
			err := r.Scan(&e.a, &e.b)
			return e, err
		})
	if err != nil {
		return nil, fmt.Errorf("concept neighbors: %w", err)
	}
	for _, e := range edges {
		out[e.a], out[e.b] = append(out[e.a], e.b), append(out[e.b], e.a)
	}
	return out, nil
}

// LastSynthesisAt is when the user's latest synthesis session (not failed) was created.
func (s *SQLiteStore) LastSynthesisAt(ctx context.Context, userID string) (*time.Time, error) {
	var at *time.Time
	err := s.c().row(ctx, `SELECT MAX(created_at) FROM check_sessions
		WHERE user_id = $1 AND kind = $2 AND status <> 'failed'`, userID, check.PurposeSynthesis).Scan(nts(&at))
	return at, err
}

// SynthesisItems is a generated synthesis set to store: integrative items
// (objective_id NULL) under the representative concept, private to the user.
type SynthesisItems struct {
	ConceptID     string
	CreatedFor    string
	PromptHash    string
	Model         string
	PromptVersion string
	Items         []check.Item
}

// SaveSynthesisItems inserts the set in one transaction, in order.
func (s *SQLiteStore) SaveSynthesisItems(ctx context.Context, d SynthesisItems) ([]CheckItem, error) {
	out := make([]CheckItem, 0, len(d.Items))
	err := s.tx(ctx, func(c conn) error {
		for _, it := range d.Items {
			rubric, err := json.Marshal(it.Rubric)
			if err != nil {
				return err
			}
			id := NewID()
			if _, err := c.exec(ctx, `
				INSERT INTO check_items (id, concept_id, objective_id, purpose, format, prompt, material, rubric,
					prompt_hash, created_for, model, prompt_version)
				VALUES ($11, $1, NULL, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
				d.ConceptID, it.Purpose, it.Format, it.Prompt, nonNullJSON(it.Material, "{}"), string(rubric),
				d.PromptHash, d.CreatedFor, d.Model, d.PromptVersion, id); err != nil {
				return fmt.Errorf("insert synthesis item: %w", err)
			}
			saved, err := scanCheckItem(c.row(ctx, `SELECT `+checkItemColumns+` FROM check_items ci WHERE ci.id = $1`, id))
			if err != nil {
				return fmt.Errorf("load synthesis item: %w", err)
			}
			out = append(out, saved)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// pullDueCovers brings the covered objectives' reviews forward after a failed
// synthesis item (learning.PullDue): box, mark and lapses stay.
func pullDueCovers(ctx context.Context, c conn, userID string, objectiveIDs []string, now time.Time, intervals []int) error {
	for _, id := range objectiveIDs {
		l, err := lockMastery(ctx, c, userID, id)
		if err != nil {
			return fmt.Errorf("load mastery: %w", err)
		}
		if !l.Exists || l.M.NextReviewAt == nil {
			continue
		}
		next := learning.PullDue(l.M, now, intervals)
		if next.NextReviewAt == nil || next.NextReviewAt.Equal(*l.M.NextReviewAt) {
			continue
		}
		if _, err := c.exec(ctx, `UPDATE user_objective_mastery SET next_review_at = $3 WHERE user_id = $1 AND objective_id = $2`,
			userID, id, next.NextReviewAt); err != nil {
			return fmt.Errorf("pull review %s: %w", id, err)
		}
	}
	return nil
}

// SynthesisObjective is a picked objective with its concept for the prompt.
type SynthesisObjective struct {
	ID, ConceptID, ConceptName, Text string
}

// SynthesisObjectives loads the picked objectives (active, visible), in the given order.
func (s *SQLiteStore) SynthesisObjectives(ctx context.Context, userID string, ids []string) ([]SynthesisObjective, error) {
	list, err := collect(s.c().query(ctx, `SELECT o.id, o.concept_id, c.name, o.text FROM objectives o
		JOIN concepts c ON c.id = o.concept_id
		WHERE o.id IN (SELECT value FROM json_each($2)) AND o.status = 'active' AND `+visibleTo("c", "$1"), userID, ids),
		func(r rowScanner) (SynthesisObjective, error) {
			var o SynthesisObjective
			err := r.Scan(&o.ID, &o.ConceptID, &o.ConceptName, &o.Text)
			return o, err
		})
	if err != nil {
		return nil, fmt.Errorf("synthesis objectives: %w", err)
	}
	byID := map[string]SynthesisObjective{}
	for _, o := range list {
		byID[o.ID] = o
	}
	out := make([]SynthesisObjective, 0, len(ids))
	for _, id := range ids {
		if o, ok := byID[id]; ok {
			out = append(out, o)
		}
	}
	return out, nil
}
