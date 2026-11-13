package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
)

const misconceptionColumns = `m.id, m.concept_id, COALESCE(c.name, ''), COALESCE(m.objective_id, ''), COALESCE(o.text, ''),
	m.text, COALESCE(m.turn_id, ''), m.status, m.created_at, m.resolved_at`

const misconceptionFrom = ` FROM user_misconceptions m
	LEFT JOIN concepts c ON c.id = m.concept_id
	LEFT JOIN objectives o ON o.id = m.objective_id`

func scanMisconception(row rowScanner) (models.Misconception, error) {
	var m models.Misconception
	err := row.Scan(&m.ID, &m.ConceptID, &m.ConceptName, &m.ObjectiveID, &m.ObjectiveText, &m.Text, &m.TurnID,
		&m.Status, ts(&m.CreatedAt), nts(&m.ResolvedAt))
	return m, err
}

// AcknowledgeMisconception stores the misconception of the user's judged turn
// ('맞아요, 그렇게 알았어요'): text from the turn's grade.misconception, concept
// and objective from its item. Acknowledging the same turn again returns the
// stored row (created=false). ErrNotFound when the turn is not the user's or
// carries no misconception.
// SQLite: RETURNING on a DO NOTHING conflict → RowsAffected; the id is store.NewID().
func (s *SQLiteStore) AcknowledgeMisconception(ctx context.Context, userID, turnID string) (models.Misconception, bool, error) {
	res, err := s.c().exec(ctx, `
		INSERT INTO user_misconceptions (id, user_id, concept_id, objective_id, text, turn_id)
		SELECT $3, t.user_id, i.concept_id, i.objective_id, trim(json_extract(t.grade, '$.misconception')), t.id
		FROM check_turns t JOIN check_items i ON i.id = t.item_id
		WHERE t.user_id = $1 AND t.id = $2 AND t.status = 'ready'
			AND trim(COALESCE(json_extract(t.grade, '$.misconception'), '')) <> ''
		ON CONFLICT (user_id, turn_id) DO NOTHING`, userID, turnID, NewID())
	if err != nil {
		return models.Misconception{}, false, fmt.Errorf("acknowledge misconception: %w", err)
	}
	n, _ := res.RowsAffected()
	m, err := scanMisconception(s.c().row(ctx, `SELECT `+misconceptionColumns+misconceptionFrom+`
		WHERE m.user_id = $1 AND m.turn_id = $2`, userID, turnID))
	if errors.Is(err, sql.ErrNoRows) {
		return m, false, ErrNotFound
	}
	if err != nil {
		return m, false, fmt.Errorf("load misconception: %w", err)
	}
	return m, n > 0, nil
}

// ListMisconceptions returns the user's misconceptions of one status, latest
// first (resolved by resolution time, open by acknowledgement time).
// NULLS LAST → `resolved_at IS NULL` first key; same timestamps by rowid.
func (s *SQLiteStore) ListMisconceptions(ctx context.Context, userID, status string, limit int) ([]models.Misconception, error) {
	order := `m.created_at DESC, m.rowid DESC`
	if status == learning.MisconceptionResolved {
		order = `m.resolved_at IS NULL, m.resolved_at DESC, m.created_at DESC, m.rowid DESC`
	}
	list, err := collect(s.c().query(ctx, `SELECT `+misconceptionColumns+misconceptionFrom+`
		WHERE m.user_id = $1 AND m.status = $2 ORDER BY `+order+` LIMIT $3`, userID, status, limit), scanMisconception)
	if err != nil {
		return nil, fmt.Errorf("misconceptions: %w", err)
	}
	return list, nil
}

// OpenMisconceptions returns up to limit open misconceptions of conceptID and
// its direct prerequisites, latest first ("" = any concept) — the prompt block
// "이 수강생이 인정한 오개념".
func (s *SQLiteStore) OpenMisconceptions(ctx context.Context, userID, conceptID string, limit int) ([]models.Misconception, error) {
	list, err := collect(s.c().query(ctx, `SELECT `+misconceptionColumns+misconceptionFrom+`
		WHERE m.user_id = $1 AND m.status = 'open' AND ($2 = '' OR m.concept_id = $2 OR m.concept_id IN (
			SELECT e.from_id FROM concept_edges e WHERE e.to_id = $2 AND e.kind = 'prerequisite'))
		ORDER BY m.created_at DESC, m.rowid DESC LIMIT $3`, userID, conceptID, limit), scanMisconception)
	if err != nil {
		return nil, fmt.Errorf("misconceptions: %w", err)
	}
	return list, nil
}

// MisconceptionTurnIDs returns the turns of a session whose misconception the user acknowledged.
func (s *SQLiteStore) MisconceptionTurnIDs(ctx context.Context, userID, sessionID string) ([]string, error) {
	ids, err := collect(s.c().query(ctx, `
		SELECT m.turn_id FROM user_misconceptions m JOIN check_turns t ON t.id = m.turn_id
		WHERE m.user_id = $1 AND t.session_id = $2 ORDER BY t.seq`, userID, sessionID), func(row rowScanner) (string, error) {
		var id string
		err := row.Scan(&id)
		return id, err
	})
	if err != nil {
		return nil, fmt.Errorf("misconception turns: %w", err)
	}
	return ids, nil
}

// resolveMisconceptions marks the objective's open misconceptions resolved —
// called inside the item-evidence transaction when the item ends unaided_correct.
func resolveMisconceptions(ctx context.Context, c conn, userID, objectiveID string) error {
	if _, err := c.exec(ctx, `UPDATE user_misconceptions SET status = 'resolved', resolved_at = NOW()
		WHERE user_id = $1 AND objective_id = $2 AND status = 'open'`, userID, objectiveID); err != nil {
		return fmt.Errorf("resolve misconceptions: %w", err)
	}
	return nil
}

// EvidencedMastery returns the active objectives of every visible concept on
// which the user has check evidence (some best ≠ none) — the input of
// learning.WeakConcept. Objectives without evidence are level 0.
func (s *SQLiteStore) EvidencedMastery(ctx context.Context, userID string) ([]models.ObjectiveMastery, error) {
	return s.objectiveMastery(ctx, userID, `o.concept_id IN (
		SELECT DISTINCT em.concept_id FROM user_objective_mastery em
		JOIN concepts c ON c.id = em.concept_id AND c.status <> 'retired' AND `+visibleTo("c", "$1")+`
		WHERE em.user_id = $2 AND em.best <> 'none')`, userID)
}
