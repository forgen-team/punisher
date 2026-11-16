package store

import (
	"context"
	"fmt"

	"github.com/forgen-team/punisher/internal/models"
)

// Link kinds.
const (
	LinkDeeperKind      = "deeper"
	LinkRegeneratedKind = "regenerated"
)

// DeeperLink is a "이 개념 더 깊게" edge to record. Exactly one of
// ChildLectureID (existing lecture) / ChildJobID (generation started or joined) is set.
type DeeperLink struct {
	UserID         string
	ParentLecture  string
	ChildLectureID string
	ChildJobID     string
	ChildConceptID string
	Anchor         string
	// Kind: deeper (default) | regenerated.
	Kind string
}

// LinkDeeper records that the user went deeper from ParentLecture. The parent
// must be readable by the user (else ErrNotFound); a link to itself or an
// identical existing link is a no-op.
func (s *SQLiteStore) LinkDeeper(ctx context.Context, l DeeperLink) error {
	if _, err := s.lectureConcept(ctx, l.UserID, l.ParentLecture); err != nil {
		return err
	}
	if l.ChildLectureID == l.ParentLecture {
		return nil
	}
	if l.Kind == "" {
		l.Kind = LinkDeeperKind
	}
	_, err := s.c().exec(ctx, `
		INSERT INTO lecture_links (user_id, parent_lecture_id, child_lecture_id, child_job_id, child_concept_id, anchor, kind)
		SELECT $1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7
		WHERE NOT EXISTS (
			SELECT 1 FROM lecture_links k
			WHERE k.user_id = $1 AND k.parent_lecture_id = $2
			  AND ((NULLIF($3, '') IS NOT NULL AND k.child_lecture_id = $3)
			    OR (NULLIF($4, '') IS NOT NULL AND k.child_job_id = $4)))`,
		l.UserID, l.ParentLecture, l.ChildLectureID, l.ChildJobID, l.ChildConceptID, l.Anchor, l.Kind)
	if err != nil {
		return fmt.Errorf("link deeper lecture: %w", err)
	}
	return nil
}

// ResolveLinkJob fills child_lecture_id of links waiting on a generation job.
func (s *SQLiteStore) ResolveLinkJob(ctx context.Context, jobID, lectureID string) error {
	_, err := s.c().exec(ctx, `
		UPDATE lecture_links SET child_lecture_id = $2
		WHERE child_job_id = $1 AND child_lecture_id IS NULL AND parent_lecture_id <> $2`, jobID, lectureID)
	if err != nil {
		return fmt.Errorf("resolve lecture link: %w", err)
	}
	return nil
}

// linkedChildSQL resolves a link's child lecture id (direct, or via the job's lecture).
const linkedChildSQL = `COALESCE(k.child_lecture_id, (SELECT jl.id FROM lectures jl WHERE jl.generation_job_id = k.child_job_id LIMIT 1))`

// LectureLinks returns the lectures the user opened via "더 깊게" from lectureID
// (children, oldest first) and the lectures it was opened from (parents).
// Only lectures the user may read are returned; unfinished/failed generations are skipped.
func (s *SQLiteStore) LectureLinks(ctx context.Context, userID, lectureID string) (parents, children []models.LinkedLecture, err error) {
	if _, err := s.lectureConcept(ctx, userID, lectureID); err != nil {
		return nil, nil, err
	}
	// First link per child (oldest), and newest link per parent — DISTINCT ON in the original.
	children, err = s.linkedLectures(ctx, `
		SELECT child, created_at, anchor, kind FROM (
			SELECT child, created_at, anchor, kind,
				ROW_NUMBER() OVER (PARTITION BY child ORDER BY created_at) AS rn
			FROM (SELECT `+linkedChildSQL+` AS child, k.created_at, k.anchor, k.kind
			      FROM lecture_links k WHERE k.user_id = $1 AND k.parent_lecture_id = $2)
			WHERE child IS NOT NULL)
		WHERE rn = 1`, userID, lectureID)
	if err != nil {
		return nil, nil, err
	}
	parents, err = s.linkedLectures(ctx, `
		SELECT child, created_at, anchor, kind FROM (
			SELECT k.parent_lecture_id AS child, k.created_at, k.anchor, k.kind,
				ROW_NUMBER() OVER (PARTITION BY k.parent_lecture_id ORDER BY k.created_at DESC) AS rn
			FROM lecture_links k
			WHERE k.user_id = $1 AND `+linkedChildSQL+` = $2)
		WHERE rn = 1`, userID, lectureID)
	return parents, children, err
}

// linkedLectures runs q (returning lecture id, linked_at, anchor, kind) and loads
// each readable lecture with the caller's progress, ordered by link time.
func (s *SQLiteStore) linkedLectures(ctx context.Context, q string, userID, lectureID string) ([]models.LinkedLecture, error) {
	out, err := collect(s.c().query(ctx, `
		WITH linked AS (`+q+`)
		SELECT l.id, l.concept_id, c.name, l.title, l.kind, l.status, l.created_at, x.created_at, x.anchor,
			COALESCE(p.percent, 0), p.completed_at, x.kind, l.prompt_version
		FROM linked x
		JOIN lectures l ON l.id = x.child AND (l.kind = 'library' OR l.requested_by = $1)
		JOIN concepts c ON c.id = l.concept_id
		LEFT JOIN user_lecture_progress p ON p.lecture_id = l.id AND p.user_id = $1
		ORDER BY x.created_at, l.id`, userID, lectureID), func(r rowScanner) (models.LinkedLecture, error) {
		var m models.LinkedLecture
		err := r.Scan(&m.ID, &m.ConceptID, &m.ConceptName, &m.Title, &m.Kind, &m.Status, ts(&m.CreatedAt),
			ts(&m.LinkedAt), &m.Anchor, &m.Percent, nts(&m.CompletedAt), &m.LinkKind, &m.PromptVersion)
		return m, err
	})
	if err != nil {
		return nil, fmt.Errorf("linked lectures: %w", err)
	}
	return out, nil
}

// parentLectureSQL is a scalar subquery: the most recent parent of lecture
// column `col` for user `$1` ("" when none). Used by list queries for grouping.
func parentLectureSQL(col string) string {
	return `COALESCE((SELECT k.parent_lecture_id FROM lecture_links k
		WHERE k.user_id = $1 AND ` + linkedChildSQL + ` = ` + col + `
		ORDER BY k.created_at DESC LIMIT 1), '')`
}
