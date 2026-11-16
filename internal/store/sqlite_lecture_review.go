package store

import (
	"context"
	"fmt"

	"github.com/forgen-team/punisher/internal/models"
)

// Backfill review of stored lectures (quality design v2.2 §2.2): published
// lectures without a pass review are reviewed a few per scheduler tick; the
// ones a learner tried to check come first.

// RequestLectureReview moves a lecture to the front of the backfill queue
// (the first request time is kept). Unknown ids are ignored.
func (s *SQLiteStore) RequestLectureReview(ctx context.Context, lectureID string) error {
	if _, err := s.c().exec(ctx, `UPDATE lectures SET review_requested_at = COALESCE(review_requested_at, NOW())
		WHERE id = $1`, lectureID); err != nil {
		return fmt.Errorf("request lecture review: %w", err)
	}
	return nil
}

// LecturesToReview returns up to limit published lectures without a pass
// review that the backfill has not reviewed yet (each lecture once — a flagged
// one is fixed by '다시 만들기', not re-reviewed every tick): requested ones
// first (oldest request), then the newest.
// NULLS LAST → `x IS NULL` first key (SQLite sorts NULL first ascending).
func (s *SQLiteStore) LecturesToReview(ctx context.Context, limit int) ([]models.Lecture, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT `+lectureColumns+` FROM lectures l
		WHERE l.status = 'published'
		  AND NOT EXISTS (SELECT 1 FROM lecture_reviews r WHERE r.lecture_id = l.id
		                  AND (r.verdict = 'pass' OR r.reviewer = 'backfill'))
		ORDER BY l.review_requested_at IS NULL, l.review_requested_at ASC,
		         l.published_at IS NULL, l.published_at DESC, l.id
		LIMIT $1`, limit), scanLecture)
	if err != nil {
		return nil, fmt.Errorf("lectures to review: %w", err)
	}
	return list, nil
}

// SetLectureReviewOutcome records a backfill review's result on the lecture:
// a pass clears the flag, a fail flags it with the reason (shown as a banner).
func (s *SQLiteStore) SetLectureReviewOutcome(ctx context.Context, lectureID string, passed bool, reason string) error {
	if passed {
		reason = ""
	}
	if _, err := s.c().exec(ctx, `UPDATE lectures SET flagged = $2, flag_reason = $3, review_requested_at = NULL
		WHERE id = $1`, lectureID, !passed, reason); err != nil {
		return fmt.Errorf("set lecture review outcome: %w", err)
	}
	return nil
}
