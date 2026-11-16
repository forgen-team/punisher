package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
)

// Exercise attempt statuses (lecture_exercise_attempts.status).
const (
	AttemptPending = "pending"
	AttemptDone    = "done"
	AttemptFailed  = "failed"
)

const attemptColumns = `id, lecture_id, anchor, content_hash, kind, answer, verdict, feedback, status, created_at`

func scanAttempt(row rowScanner) (models.ExerciseAttempt, error) {
	var a models.ExerciseAttempt
	err := row.Scan(&a.ID, &a.LectureID, &a.Anchor, &a.ContentHash, &a.Kind, &a.Answer, &a.Verdict, &a.Feedback, &a.Status, ts(&a.CreatedAt))
	return a, err
}

// NewExerciseAttempt is an attempt to save; Status "" = pending.
type NewExerciseAttempt struct {
	UserID, LectureID, Anchor, ContentHash, Section, Kind, Answer, Verdict, Status string
}

// InsertExerciseAttempt saves an attempt (a skip is saved done with verdict skipped).
func (s *SQLiteStore) InsertExerciseAttempt(ctx context.Context, a NewExerciseAttempt) (models.ExerciseAttempt, error) {
	if a.Status == "" {
		a.Status = AttemptPending
	}
	out, err := scanAttempt(s.c().row(ctx, `
		INSERT INTO lecture_exercise_attempts (id, user_id, lecture_id, anchor, content_hash, section, kind, answer, verdict, status, created_at)
		VALUES ($10, $1, $2, $3, $4, $5, $6, $7, $8, $9, $11)
		RETURNING `+attemptColumns, a.UserID, a.LectureID, a.Anchor, a.ContentHash, a.Section, a.Kind, a.Answer, a.Verdict, a.Status,
		NewID(), clockNow()))
	if err != nil {
		return out, fmt.Errorf("insert exercise attempt: %w", err)
	}
	return out, nil
}

// FinishExerciseAttempt records the grading of a pending attempt (status done or failed).
func (s *SQLiteStore) FinishExerciseAttempt(ctx context.Context, userID, id, status, verdict, feedback, model string) error {
	_, err := s.c().exec(ctx, `UPDATE lecture_exercise_attempts SET status = $3, verdict = $4, feedback = $5, model = $6
		WHERE user_id = $1 AND id = $2 AND status = 'pending'`, userID, id, status, verdict, feedback, model)
	if err != nil {
		return fmt.Errorf("finish exercise attempt: %w", err)
	}
	return nil
}

// LatestExerciseAttempts returns the user's newest attempt per (anchor, content hash) of a lecture.
// DISTINCT ON → ROW_NUMBER (same created_at: the later rowid).
func (s *SQLiteStore) LatestExerciseAttempts(ctx context.Context, userID, lectureID string) ([]models.ExerciseAttempt, error) {
	list, err := collect(s.c().query(ctx, `SELECT `+attemptColumns+` FROM (
			SELECT *, ROW_NUMBER() OVER (PARTITION BY anchor, content_hash ORDER BY created_at DESC, rowid DESC) AS rn
			FROM lecture_exercise_attempts WHERE user_id = $1 AND lecture_id = $2)
		WHERE rn = 1 ORDER BY anchor, content_hash`, userID, lectureID), scanAttempt)
	if err != nil {
		return nil, fmt.Errorf("list exercise attempts: %w", err)
	}
	if list == nil {
		list = []models.ExerciseAttempt{}
	}
	return list, nil
}

// GetExerciseAttempt returns one of the user's attempts.
func (s *SQLiteStore) GetExerciseAttempt(ctx context.Context, userID, id string) (models.ExerciseAttempt, error) {
	a, err := scanAttempt(s.c().row(ctx, `SELECT `+attemptColumns+` FROM lecture_exercise_attempts
		WHERE user_id = $1 AND id = $2`, userID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
}

// DeleteExerciseAttempt removes one of the user's attempts (e.g. a pasted lab output).
func (s *SQLiteStore) DeleteExerciseAttempt(ctx context.Context, userID, id string) error {
	res, err := s.c().exec(ctx, `DELETE FROM lecture_exercise_attempts WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return fmt.Errorf("delete exercise attempt: %w", err)
	}
	if affected(res) == 0 {
		return ErrNotFound
	}
	return nil
}

// LectureSignals counts, per section, the user's latest exercise/lab attempt
// per block and the lecture questions asked there (v3 §6).
func (s *SQLiteStore) LectureSignals(ctx context.Context, userID, lectureID string) ([]learning.SectionSignal, error) {
	bySection := map[string]*learning.SectionSignal{}
	var order []string
	get := func(section string) *learning.SectionSignal {
		if bySection[section] == nil {
			bySection[section] = &learning.SectionSignal{Section: section}
			order = append(order, section)
		}
		return bySection[section]
	}
	attempts, err := s.LatestExerciseAttempts(ctx, userID, lectureID)
	if err != nil {
		return nil, err
	}
	for _, a := range attempts {
		if a.Status != AttemptDone {
			continue
		}
		sig := get(learning.SectionOfAnchor(a.Anchor))
		miss := learning.AttemptMiss(a.Verdict)
		if a.Kind == learning.AttemptLab {
			sig.Labs++
			if miss {
				sig.LabMisses++
			}
		} else {
			sig.Exercises++
			if miss {
				sig.ExerciseMs++
			}
		}
	}
	type count struct {
		anchor string
		n      int
	}
	counts, err := collect(s.c().query(ctx, `SELECT anchor, COUNT(*) FROM lecture_questions
		WHERE user_id = $1 AND lecture_id = $2 AND anchor <> '' GROUP BY anchor`, userID, lectureID), func(r rowScanner) (count, error) {
		var c count
		err := r.Scan(&c.anchor, &c.n)
		return c, err
	})
	if err != nil {
		return nil, fmt.Errorf("lecture question signals: %w", err)
	}
	for _, c := range counts {
		get(learning.SectionOfAnchor(c.anchor)).Questions += c.n
	}
	out := make([]learning.SectionSignal, 0, len(order))
	for _, sec := range order {
		out = append(out, *bySection[sec])
	}
	return out, nil
}
