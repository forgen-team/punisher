package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/forgen-team/punisher/internal/models"
)

const noteColumns = `id, COALESCE(lecture_id, ''), COALESCE(concept_id, ''), anchor, kind, quote, body, created_at, updated_at`

func scanNote(row rowScanner) (models.Note, error) {
	var n models.Note
	err := row.Scan(&n.ID, &n.LectureID, &n.ConceptID, &n.Anchor, &n.Kind, &n.Quote, &n.Body, ts(&n.CreatedAt), ts(&n.UpdatedAt))
	return n, err
}

// ListNotes filters by lecture and/or concept ("" = any).
func (s *SQLiteStore) ListNotes(ctx context.Context, userID, lectureID, conceptID string) ([]models.Note, error) {
	notes, err := collect(s.c().query(ctx, `
		SELECT `+noteColumns+` FROM notes
		WHERE user_id = $1 AND ($2 = '' OR lecture_id = $2) AND ($3 = '' OR concept_id = $3)
		ORDER BY created_at`, userID, lectureID, conceptID), scanNote)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	return notes, nil
}

// CreateNote inserts a note.
func (s *SQLiteStore) CreateNote(ctx context.Context, userID string, n models.Note) (models.Note, error) {
	return scanNote(s.c().row(ctx, `
		INSERT INTO notes (id, user_id, lecture_id, concept_id, anchor, kind, quote, body)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7, $8)
		RETURNING `+noteColumns, NewID(), userID, n.LectureID, n.ConceptID, n.Anchor, n.Kind, n.Quote, n.Body))
}

// UpdateNote changes anchor/quote/body of the user's note.
func (s *SQLiteStore) UpdateNote(ctx context.Context, userID, id string, anchor, quote, body *string) (models.Note, error) {
	n, err := scanNote(s.c().row(ctx, `
		UPDATE notes SET anchor = COALESCE($3, anchor), quote = COALESCE($4, quote),
			body = COALESCE($5, body), updated_at = NOW()
		WHERE id = $2 AND user_id = $1
		RETURNING `+noteColumns, userID, id, anchor, quote, body))
	if errors.Is(err, sql.ErrNoRows) {
		return n, ErrNotFound
	}
	return n, err
}

// DeleteNote removes the user's note.
func (s *SQLiteStore) DeleteNote(ctx context.Context, userID, id string) error {
	res, err := s.c().exec(ctx, `DELETE FROM notes WHERE id = $2 AND user_id = $1`, userID, id)
	if err != nil {
		return err
	}
	if affected(res) == 0 {
		return ErrNotFound
	}
	return nil
}

const goalColumns = `id, title, target_concept_ids, status, created_at, updated_at`

func scanGoal(row rowScanner) (models.Goal, error) {
	var g models.Goal
	err := row.Scan(&g.ID, &g.Title, strs(&g.TargetConceptIDs), &g.Status, ts(&g.CreatedAt), ts(&g.UpdatedAt))
	return g, err
}

// ListGoals returns goals, active first then most recently updated.
func (s *SQLiteStore) ListGoals(ctx context.Context, userID, status string) ([]models.Goal, error) {
	goals, err := collect(s.c().query(ctx, `
		SELECT `+goalColumns+` FROM user_goals
		WHERE user_id = $1 AND ($2 = '' OR status = $2)
		ORDER BY (status = 'active') DESC, updated_at DESC`, userID, status), scanGoal)
	if err != nil {
		return nil, fmt.Errorf("list goals: %w", err)
	}
	return goals, nil
}

// GetGoal loads one goal.
func (s *SQLiteStore) GetGoal(ctx context.Context, userID, id string) (models.Goal, error) {
	g, err := scanGoal(s.c().row(ctx,
		`SELECT `+goalColumns+` FROM user_goals WHERE id = $2 AND user_id = $1`, userID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return g, ErrNotFound
	}
	return g, err
}

// CreateGoal inserts an active goal.
func (s *SQLiteStore) CreateGoal(ctx context.Context, userID, title string, targets []string) (models.Goal, error) {
	return scanGoal(s.c().row(ctx, `
		INSERT INTO user_goals (id, user_id, title, target_concept_ids) VALUES ($1, $2, $3, $4)
		RETURNING `+goalColumns, NewID(), userID, title, nonNil(targets)))
}

// UpdateGoal changes title/targets/status (nil = unchanged).
func (s *SQLiteStore) UpdateGoal(ctx context.Context, userID, id string, title *string, targets []string, status *string) (models.Goal, error) {
	var targetsArg any // NULL keeps the current targets
	if targets != nil {
		targetsArg = targets
	}
	g, err := scanGoal(s.c().row(ctx, `
		UPDATE user_goals SET title = COALESCE($3, title),
			target_concept_ids = COALESCE($4, target_concept_ids),
			status = COALESCE($5, status), updated_at = NOW()
		WHERE id = $2 AND user_id = $1
		RETURNING `+goalColumns, userID, id, title, targetsArg, status))
	if errors.Is(err, sql.ErrNoRows) {
		return g, ErrNotFound
	}
	return g, err
}

// DeleteGoal removes a goal.
func (s *SQLiteStore) DeleteGoal(ctx context.Context, userID, id string) error {
	res, err := s.c().exec(ctx, `DELETE FROM user_goals WHERE id = $2 AND user_id = $1`, userID, id)
	if err != nil {
		return err
	}
	if affected(res) == 0 {
		return ErrNotFound
	}
	return nil
}
