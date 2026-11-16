package store

import (
	"context"
	"fmt"

	"github.com/forgen-team/punisher/internal/learning"
)

// DismissLecture hides an in-progress lecture from 이어보기 (soft: the lecture
// and its progress are kept; RestoreLecture or reading it again undoes it).
// When no other visible in-progress lecture of the same concept remains, an
// in_progress concept state (set only by reading) is cleared so a mis-tapped
// lecture does not leave the concept "진행 중" on the map. completed/known
// states are never touched.
func (s *SQLiteStore) DismissLecture(ctx context.Context, userID, lectureID string) error {
	conceptID, err := s.lectureConcept(ctx, userID, lectureID)
	if err != nil {
		return err
	}
	return s.tx(ctx, func(c conn) error {
		res, err := c.exec(ctx, `
			UPDATE user_lecture_progress SET dismissed_at = COALESCE(dismissed_at, NOW())
			WHERE user_id = $1 AND lecture_id = $2 AND completed_at IS NULL`, userID, lectureID)
		if err != nil {
			return fmt.Errorf("dismiss lecture: %w", err)
		}
		if affected(res) == 0 {
			return ErrNotFound
		}
		if _, err := c.exec(ctx, `
			DELETE FROM user_concept_states
			WHERE user_id = $1 AND concept_id = $2 AND state = $3
			  AND NOT EXISTS (
			    SELECT 1 FROM user_lecture_progress p JOIN lectures l ON l.id = p.lecture_id
			    WHERE p.user_id = $1 AND l.concept_id = $2 AND p.dismissed_at IS NULL)`,
			userID, conceptID, learning.InProgress); err != nil {
			return fmt.Errorf("clear concept state: %w", err)
		}
		return nil
	})
}

// RestoreLecture undoes DismissLecture and marks the concept in progress again.
func (s *SQLiteStore) RestoreLecture(ctx context.Context, userID, lectureID string) error {
	conceptID, err := s.lectureConcept(ctx, userID, lectureID)
	if err != nil {
		return err
	}
	res, err := s.c().exec(ctx, `
		UPDATE user_lecture_progress SET dismissed_at = NULL
		WHERE user_id = $1 AND lecture_id = $2`, userID, lectureID)
	if err != nil {
		return fmt.Errorf("restore lecture: %w", err)
	}
	if affected(res) == 0 {
		return ErrNotFound
	}
	_, err = s.SetConceptState(ctx, userID, conceptID, learning.InProgress)
	return err
}

// DismissedConceptIDs returns concepts whose only progress is dismissed lectures
// (no visible in-progress or completed lecture) — "dismissed" in per-area summaries.
func (s *SQLiteStore) DismissedConceptIDs(ctx context.Context, userID string) (map[string]bool, error) {
	ids, err := collect(s.c().query(ctx, `
		SELECT l.concept_id FROM user_lecture_progress p JOIN lectures l ON l.id = p.lecture_id
		WHERE p.user_id = $1
		GROUP BY l.concept_id
		HAVING MIN(p.dismissed_at IS NOT NULL) = 1`, userID), scanString)
	if err != nil {
		return nil, fmt.Errorf("dismissed concepts: %w", err)
	}
	out := map[string]bool{}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}
