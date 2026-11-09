package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/wooo-jin/punisher/internal/models"
)

const questionColumns = `id, COALESCE(lecture_id, ''), COALESCE(concept_id, ''), anchor, kind,
	question, answer, status, model, error, linked_concept_ids, proposed_concepts, created_at, answered_at`

func scanQuestion(row rowScanner) (models.Question, error) {
	var q models.Question
	err := row.Scan(&q.ID, &q.LectureID, &q.ConceptID, &q.Anchor, &q.Kind, &q.Question, &q.Answer, &q.Status,
		&q.Model, &q.Error, strs(&q.LinkedConceptIDs), rawJSON(&q.ProposedConcepts), ts(&q.CreatedAt), nts(&q.AnsweredAt))
	return q, err
}

// CreateQuestion inserts a pending question (or alt_explanation request).
func (s *SQLiteStore) CreateQuestion(ctx context.Context, userID string, q models.Question) (models.Question, error) {
	return scanQuestion(s.c().row(ctx, `
		INSERT INTO lecture_questions (id, user_id, lecture_id, concept_id, anchor, kind, question, status)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7, 'pending')
		RETURNING `+questionColumns, NewID(), userID, q.LectureID, q.ConceptID, q.Anchor, q.Kind, q.Question))
}

// AnswerQuestion records a successful answer.
func (s *SQLiteStore) AnswerQuestion(ctx context.Context, id, answer, model string, linked []string, proposed json.RawMessage) (models.Question, error) {
	return scanQuestion(s.c().row(ctx, `
		UPDATE lecture_questions SET answer = $2, status = 'answered', model = $3, linked_concept_ids = $4,
			proposed_concepts = $5, error = '', answered_at = NOW()
		WHERE id = $1 RETURNING `+questionColumns, id, answer, model, nonNil(linked), nonNullJSON(proposed, "[]")))
}

// FailQuestion records an LLM failure (never stored as answered).
func (s *SQLiteStore) FailQuestion(ctx context.Context, id, model, errMsg string) (models.Question, error) {
	return scanQuestion(s.c().row(ctx, `
		UPDATE lecture_questions SET status = 'failed', model = $2, error = $3
		WHERE id = $1 RETURNING `+questionColumns, id, model, truncateText(errMsg, 2000)))
}

// ListQuestions returns the user's thread for a lecture, oldest first.
func (s *SQLiteStore) ListQuestions(ctx context.Context, userID, lectureID string) ([]models.Question, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT `+questionColumns+` FROM lecture_questions
		WHERE user_id = $1 AND lecture_id = $2 ORDER BY created_at`, userID, lectureID), scanQuestion)
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	return list, nil
}

// GetQuestion loads one of the user's questions.
func (s *SQLiteStore) GetQuestion(ctx context.Context, userID, id string) (models.Question, error) {
	q, err := scanQuestion(s.c().row(ctx,
		`SELECT `+questionColumns+` FROM lecture_questions WHERE id = $2 AND user_id = $1`, userID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return q, ErrNotFound
	}
	return q, err
}
