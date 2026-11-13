package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
)

// ListCheckHistory returns the user's check, review and '이미 앎' verification
// (known_verify) sessions, newest first (kind "" = all three; the pretest and
// placement diagnostics are not history entries). next is "" on the last page.
func (s *SQLiteStore) ListCheckHistory(ctx context.Context, userID, kind string, limit int, cursor string) ([]models.CheckHistoryItem, string, error) {
	cur, err := decodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	items, err := collect(s.c().query(ctx, `
		SELECT cs.id, cs.kind, cs.grader, cs.status, COALESCE(cs.concept_id, ''), COALESCE(c.name, ''),
			COALESCE(cs.lecture_id, ''), COALESCE(l.title, ''), cs.session_key,
			json_array_length(cs.item_ids), COALESCE(json_array_length(cs.result, '$.items'), 0),
			(SELECT COUNT(*) FROM json_each(cs.result, '$.items') e
			 WHERE json_extract(e.value, '$.outcome') = 'unaided_correct'),
			cs.created_at, cs.finished_at
		FROM check_sessions cs
		LEFT JOIN concepts c ON c.id = cs.concept_id
		LEFT JOIN lectures l ON l.id = cs.lecture_id
		WHERE cs.user_id = $1 AND cs.kind IN ('check', 'review', 'known_verify') AND ($2 = '' OR cs.kind = $2) AND (cs.created_at, cs.id) < ($3, $4)
		ORDER BY cs.created_at DESC, cs.id DESC LIMIT $5`, userID, kind, cur.At, cur.ID, limit+1),
		func(row rowScanner) (models.CheckHistoryItem, error) {
			var h models.CheckHistoryItem
			var key string
			err := row.Scan(&h.ID, &h.Kind, &h.Grader, &h.Status, &h.ConceptID, &h.ConceptName, &h.LectureID, &h.LectureTitle,
				&key, &h.ItemCount, &h.DoneCount, &h.UnaidedCorrect, ts(&h.CreatedAt), nts(&h.FinishedAt))
			if day, ok := strings.CutPrefix(key, "review:"); ok {
				h.ReviewDay = day
			}
			return h, err
		})
	if err != nil {
		return nil, "", fmt.Errorf("list check history: %w", err)
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		next = encodeCursor(last.CreatedAt, last.ID)
	}
	return items, next, nil
}

// RetentionSamples returns, for review items finished since t on items the
// user had not seen (reused_item = false), the grader, the state before the
// review (grade.prev) and whether the first graded answer was correct
// (design §2.7). Items without a graded answer (held, revealed unanswered) and
// '이미 앎' verification items mixed into reviews (not a retention test) are left out.
// JOIN LATERAL (… LIMIT 1) → a correlated scalar subquery, kept only when present.
func (s *SQLiteStore) RetentionSamples(ctx context.Context, userID string, since time.Time) ([]learning.RetentionSample, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT grader, prev_box, prev_step, first_verdict = 'correct' FROM (
			SELECT j.grader, COALESCE(json_extract(j.grade, '$.prev.box'), 0) AS prev_box,
				COALESCE(json_extract(j.grade, '$.prev.gradStep'), 0) AS prev_step,
				(SELECT f.verdict FROM check_turns f
				 WHERE f.session_id = j.session_id AND f.item_id = j.item_id AND f.status = 'ready' AND f.move <> 'dispute'
				   AND f.verdict IN ('correct', 'partial', 'incorrect')
				 ORDER BY f.seq LIMIT 1) AS first_verdict
			FROM check_turns j
			JOIN check_sessions cs ON cs.id = j.session_id AND cs.kind = 'review'
			JOIN check_items ci ON ci.id = j.item_id AND ci.purpose <> 'known_verify'
			WHERE j.user_id = $1 AND j.outcome IS NOT NULL AND j.created_at >= $2 AND NOT j.reused_item
			  AND j.grader IN ('llm', 'self'))
		WHERE first_verdict IS NOT NULL`, userID, since), func(row rowScanner) (learning.RetentionSample, error) {
		var r learning.RetentionSample
		err := row.Scan(&r.Grader, &r.PrevBox, &r.PrevGradStep, &r.Correct)
		return r, err
	})
	if err != nil {
		return nil, fmt.Errorf("retention samples: %w", err)
	}
	return list, nil
}
