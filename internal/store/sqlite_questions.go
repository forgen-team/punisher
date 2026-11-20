package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
)

const questionColumns = `id, COALESCE(lecture_id, ''), COALESCE(concept_id, ''), anchor, kind,
	question, answer, status, model, error, linked_concept_ids, proposed_concepts, created_at, answered_at,
	quote, COALESCE(parent_id, ''), mode`

func scanQuestion(row rowScanner) (models.Question, error) {
	var q models.Question
	err := row.Scan(&q.ID, &q.LectureID, &q.ConceptID, &q.Anchor, &q.Kind, &q.Question, &q.Answer, &q.Status,
		&q.Model, &q.Error, strs(&q.LinkedConceptIDs), rawJSON(&q.ProposedConcepts), ts(&q.CreatedAt), nts(&q.AnsweredAt),
		&q.Quote, &q.ParentID, &q.Mode)
	return q, err
}

// CreateQuestion inserts a pending question (or alt_explanation request).
// Thread fields: Quote, ParentID ("" = root) and Mode ("" = question).
func (s *SQLiteStore) CreateQuestion(ctx context.Context, userID string, q models.Question) (models.Question, error) {
	return insertQuestion(ctx, s.c(), userID, q)
}

func insertQuestion(ctx context.Context, c conn, userID string, q models.Question) (models.Question, error) {
	return scanQuestion(c.row(ctx, `
		INSERT INTO lecture_questions (id, user_id, lecture_id, concept_id, anchor, kind, question, status, quote, parent_id, mode)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7, 'pending', $8, NULLIF($9, ''), COALESCE(NULLIF($10, ''), 'question'))
		RETURNING `+questionColumns, NewID(), userID, q.LectureID, q.ConceptID, q.Anchor, q.Kind, q.Question,
		q.Quote, q.ParentID, q.Mode))
}

// CreateThreadQuestion inserts a reading-thread entry like CreateQuestion, but
// a non-failed entry with the same thread key created within dedupeWindow is
// returned instead (double submit; created=false): for a root (ParentID "")
// the same lecture, quote, mode and question; for a follow-up the same parent,
// mode and question. A failed entry is never reused, so a retry asks again.
// The check and the insert are one transaction (like CreateInquiry).
func (s *SQLiteStore) CreateThreadQuestion(ctx context.Context, userID string, q models.Question, dedupeWindow time.Duration) (out models.Question, created bool, err error) {
	mode := q.Mode
	if mode == "" {
		mode = learning.ThreadQuestion
	}
	since := time.Now().Add(-dedupeWindow)
	err = s.tx(ctx, func(c conn) error {
		var row *sql.Row
		if q.ParentID == "" {
			row = c.row(ctx, `SELECT `+questionColumns+` FROM lecture_questions
				WHERE user_id = $1 AND kind = 'question' AND parent_id IS NULL AND lecture_id = $2 AND quote = $3
				  AND mode = $4 AND question = $5 AND status <> 'failed' AND created_at > $6
				ORDER BY created_at DESC LIMIT 1`, userID, q.LectureID, q.Quote, mode, q.Question, since)
		} else {
			row = c.row(ctx, `SELECT `+questionColumns+` FROM lecture_questions
				WHERE user_id = $1 AND kind = 'question' AND parent_id = $2
				  AND mode = $3 AND question = $4 AND status <> 'failed' AND created_at > $5
				ORDER BY created_at DESC LIMIT 1`, userID, q.ParentID, mode, q.Question, since)
		}
		existing, err := scanQuestion(row)
		if err == nil {
			out = existing
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("find recent thread entry: %w", err)
		}
		if out, err = insertQuestion(ctx, c, userID, q); err != nil {
			return fmt.Errorf("create thread entry: %w", err)
		}
		created = true
		return nil
	})
	return out, created, err
}

// ListThread returns a reading thread — the root and its follow-ups, oldest
// first. ErrNotFound when rootID is not one of the user's questions.
func (s *SQLiteStore) ListThread(ctx context.Context, userID, rootID string) ([]models.Question, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT `+questionColumns+` FROM lecture_questions
		WHERE user_id = $1 AND (id = $2 OR parent_id = $2)
		ORDER BY (id = $2) DESC, created_at, id`, userID, rootID), scanQuestion)
	if err != nil {
		return nil, fmt.Errorf("list thread: %w", err)
	}
	if len(list) == 0 || list[0].ID != rootID {
		return nil, ErrNotFound
	}
	return list, nil
}

// EasierRequestsSince counts the user's answered '더 쉽게' thread entries
// (roots and follow-ups) created at or after since — the level-suggestion
// signal. Failed and pending entries (budget, outage, double taps) do not count.
func (s *SQLiteStore) EasierRequestsSince(ctx context.Context, userID string, since time.Time) (int, error) {
	var n int
	if err := s.c().row(ctx, `SELECT COUNT(*) FROM lecture_questions
		WHERE user_id = $1 AND mode = $2 AND status = 'answered' AND created_at >= $3`,
		userID, learning.ThreadEasier, since).Scan(&n); err != nil {
		return 0, fmt.Errorf("count easier requests: %w", err)
	}
	return n, nil
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
