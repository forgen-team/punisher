package store

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wooo-jin/punisher/internal/models"
)

// ErrInvalidCursor is returned for malformed pagination cursors.
var ErrInvalidCursor = errors.New("invalid cursor")

// pageCursor is a keyset position: rows strictly before (At, ID) in the sort order.
type pageCursor struct {
	At time.Time
	ID string
}

func encodeCursor(at time.Time, id string) string {
	raw := at.UTC().Format(time.RFC3339Nano) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(cursor string) (pageCursor, error) {
	if cursor == "" {
		return pageCursor{At: time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC), ID: "ffffffff-ffff-ffff-ffff-ffffffffffff"}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return pageCursor{}, ErrInvalidCursor
	}
	atText, id, found := strings.Cut(string(raw), "|")
	if !found || id == "" {
		return pageCursor{}, ErrInvalidCursor
	}
	at, err := time.Parse(time.RFC3339Nano, atText)
	if err != nil {
		return pageCursor{}, ErrInvalidCursor
	}
	return pageCursor{At: at, ID: id}, nil
}

// ListMyQuestions returns the caller's questions across lectures, newest first.
// conceptID ("" = any) matches the question's concept or, if unset, its lecture's concept.
// next is "" on the last page.
func (s *SQLiteStore) ListMyQuestions(ctx context.Context, userID, conceptID string, limit int, cursor string) ([]models.MyQuestion, string, error) {
	cur, err := decodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	items, err := collect(s.c().query(ctx, `
		SELECT q.id, COALESCE(q.lecture_id, ''), COALESCE(q.concept_id, l.concept_id, ''), q.anchor, q.kind,
			q.question, q.answer, q.status, q.model, q.error, q.linked_concept_ids, q.proposed_concepts,
			q.created_at, q.answered_at, COALESCE(l.title, ''), COALESCE(c.name, '')
		FROM lecture_questions q
		LEFT JOIN lectures l ON l.id = q.lecture_id
		LEFT JOIN concepts c ON c.id = COALESCE(q.concept_id, l.concept_id)
		WHERE q.user_id = $1 AND ($2 = '' OR COALESCE(q.concept_id, l.concept_id) = $2)
		  AND (q.created_at, q.id) < ($3, $4)
		ORDER BY q.created_at DESC, q.id DESC LIMIT $5`,
		userID, conceptID, cur.At, cur.ID, limit+1), func(row rowScanner) (models.MyQuestion, error) {
		var m models.MyQuestion
		q := &m.Question
		err := row.Scan(&q.ID, &q.LectureID, &q.ConceptID, &q.Anchor, &q.Kind, &q.Question, &q.Answer, &q.Status,
			&q.Model, &q.Error, strs(&q.LinkedConceptIDs), rawJSON(&q.ProposedConcepts), ts(&q.CreatedAt), nts(&q.AnsweredAt),
			&m.LectureTitle, &m.ConceptName)
		return m, err
	})
	if err != nil {
		return nil, "", fmt.Errorf("list my questions: %w", err)
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		next = encodeCursor(last.CreatedAt, last.ID)
	}
	return items, next, nil
}

// ListMyLectures returns lectures the caller completed (status "completed", by
// completion time) or is reading (status "in_progress", by last visit, excluding
// lectures dismissed from 이어보기), newest first.
func (s *SQLiteStore) ListMyLectures(ctx context.Context, userID, status string, limit int, cursor string) ([]models.MyLecture, string, error) {
	cur, err := decodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	completed := status == "completed"
	sortCol, where := "p.last_seen_at", "p.completed_at IS NULL AND p.dismissed_at IS NULL"
	if completed {
		sortCol, where = "p.completed_at", "p.completed_at IS NOT NULL"
	}
	type row struct {
		models.MyLecture
		sortAt time.Time
	}
	items, err := collect(s.c().query(ctx, `
		SELECT l.id, l.concept_id, c.name, l.title, l.depth, l.kind, l.estimated_minutes,
			p.percent, p.started_at, p.last_seen_at, p.completed_at, `+sortCol+`
		FROM user_lecture_progress p
		JOIN lectures l ON l.id = p.lecture_id AND (l.kind = 'library' OR l.requested_by = p.user_id)
		JOIN concepts c ON c.id = l.concept_id
		WHERE p.user_id = $1 AND `+where+` AND (`+sortCol+`, l.id) < ($2, $3)
		ORDER BY `+sortCol+` DESC, l.id DESC LIMIT $4`,
		userID, cur.At, cur.ID, limit+1), func(r rowScanner) (row, error) {
		var m row
		err := r.Scan(&m.ID, &m.ConceptID, &m.ConceptName, &m.Title, &m.Depth, &m.Kind, &m.EstimatedMinutes,
			&m.Percent, ts(&m.StartedAt), ts(&m.LastSeenAt), nts(&m.CompletedAt), ts(&m.sortAt))
		return m, err
	})
	if err != nil {
		return nil, "", fmt.Errorf("list my lectures: %w", err)
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		next = encodeCursor(last.sortAt, last.ID)
	}
	out := make([]models.MyLecture, len(items))
	for i, it := range items {
		out[i] = it.MyLecture
	}
	return out, next, nil
}

// ListConceptLectures returns the published lectures of a concept the caller may
// read (library, or the caller's own instant lectures) with the caller's progress.
// Only the newest version per (objective set, depth) is returned.
func (s *SQLiteStore) ListConceptLectures(ctx context.Context, userID, conceptID string) ([]models.ConceptLecture, error) {
	var exists bool
	if err := s.c().row(ctx, `SELECT EXISTS (SELECT 1 FROM concepts WHERE id = $1)`, conceptID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check concept: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}
	list, err := collect(s.c().query(ctx, `
		SELECT id, title, depth, kind, version, estimated_minutes, published_at,
			CASE WHEN completed_at IS NOT NULL THEN 'completed' WHEN started_at IS NOT NULL THEN 'in_progress'
				ELSE 'not_started' END, percent
		FROM (
			SELECT l.id AS id, l.title, l.depth, l.kind, l.version, l.estimated_minutes, l.published_at,
				p.completed_at, p.started_at, COALESCE(p.percent, 0) AS percent, l.created_at,
				ROW_NUMBER() OVER (
					PARTITION BY l.objective_key, l.depth
					ORDER BY (l.kind = 'library') DESC, l.version DESC, l.created_at DESC
				) AS rn
			FROM lectures l
			LEFT JOIN user_lecture_progress p ON p.lecture_id = l.id AND p.user_id = $2
			WHERE l.concept_id = $1 AND l.status = 'published' AND (l.kind = 'library' OR l.requested_by = $2)
		) latest
		WHERE rn = 1
		ORDER BY depth, created_at DESC LIMIT 50`, conceptID, userID), func(r rowScanner) (models.ConceptLecture, error) {
		var m models.ConceptLecture
		err := r.Scan(&m.ID, &m.Title, &m.Depth, &m.Kind, &m.Version, &m.EstimatedMinutes, nts(&m.PublishedAt),
			&m.ProgressState, &m.Percent)
		return m, err
	})
	if err != nil {
		return nil, fmt.Errorf("list concept lectures: %w", err)
	}
	return list, nil
}
