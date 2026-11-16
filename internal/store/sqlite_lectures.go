package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/lecture"
	"github.com/forgen-team/punisher/internal/models"
)

const lectureColumns = `l.id, l.concept_id, l.objective_ids, l.objective_key, l.depth, l.kind, l.status,
	l.version, l.title, l.content, l.content_hash, l.sources, l.model, l.prompt_version,
	COALESCE(l.requested_by, ''), l.flagged, l.flag_reason, l.estimated_minutes,
	COALESCE(l.generation_job_id, ''), l.created_at, l.published_at, l.minutes_estimated,
	EXISTS (SELECT 1 FROM lecture_reviews r WHERE r.lecture_id = l.id AND r.verdict = 'pass')`

func scanLecture(row rowScanner) (models.Lecture, error) {
	var l models.Lecture
	var minutesEstimated bool
	err := row.Scan(&l.ID, &l.ConceptID, strs(&l.ObjectiveIDs), &l.ObjectiveKey, &l.Depth, &l.Kind, &l.Status,
		&l.Version, &l.Title, rawJSON(&l.Content), &l.ContentHash, rawJSON(&l.Sources), &l.Model, &l.PromptVersion,
		&l.RequestedBy, &l.Flagged, &l.FlagReason, &l.EstimatedMinutes,
		&l.GenerationJobID, ts(&l.CreatedAt), nts(&l.PublishedAt), &minutesEstimated, &l.Reviewed)
	if err != nil {
		return l, err
	}
	l.ReviewState = models.LectureReviewState(l.Reviewed, l.Flagged)
	if !minutesEstimated {
		l.EstimatedMinutes = contentMinutes(l.Content, l.EstimatedMinutes)
	}
	return l, nil
}

// contentMinutes is learning.EstimateMinutes of stored content (lectures saved
// before the estimate recorded section sums); fallback when it does not decode.
func contentMinutes(raw []byte, fallback int) int {
	var c lecture.Content
	if len(raw) == 0 || json.Unmarshal(raw, &c) != nil {
		return fallback
	}
	return learning.EstimateMinutes(c)
}

// NewLecture is the insert payload. Version is assigned by InsertLecture;
// EstimatedMinutes must be learning.EstimateMinutes of the content.
type NewLecture struct {
	ConceptID        string
	ObjectiveIDs     []string
	ObjectiveKey     string
	Depth            int
	Kind             string
	Status           string
	Title            string
	Content          json.RawMessage
	ContentHash      string
	Sources          json.RawMessage
	Model            string
	PromptVersion    string
	RequestedBy      string
	Flagged          bool
	FlagReason       string
	EstimatedMinutes int
	GenerationJobID  string
}

// InsertLecture stores an immutable lecture as the next version of its
// (concept, objective_key, depth) target.
func (s *SQLiteStore) InsertLecture(ctx context.Context, l NewLecture) (models.Lecture, error) {
	var requestedBy, jobID any
	if l.RequestedBy != "" {
		requestedBy = l.RequestedBy
	}
	if l.GenerationJobID != "" {
		jobID = l.GenerationJobID
	}
	id := NewID()
	var saved models.Lecture
	err := s.tx(ctx, func(c conn) error {
		if _, err := c.exec(ctx, `
			INSERT INTO lectures (id, concept_id, objective_ids, objective_key, depth, kind, status, version,
				title, content, content_hash, sources, model, prompt_version, requested_by,
				flagged, flag_reason, estimated_minutes, generation_job_id, published_at, minutes_estimated)
			SELECT $1, $2, $3, $4, $5, $6, $7,
				(SELECT COALESCE(MAX(version), 0) + 1 FROM lectures WHERE concept_id = $2 AND objective_key = $4 AND depth = $5),
				$8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18,
				CASE WHEN $7 = 'published' THEN NOW() END, 1`,
			id, l.ConceptID, nonNil(l.ObjectiveIDs), l.ObjectiveKey, l.Depth, l.Kind, l.Status,
			l.Title, nonNullJSON(l.Content, "{}"), l.ContentHash, nonNullJSON(l.Sources, "[]"), l.Model, l.PromptVersion,
			requestedBy, l.Flagged, l.FlagReason, l.EstimatedMinutes, jobID); err != nil {
			return err
		}
		var err error
		saved, err = scanLecture(c.row(ctx, `SELECT `+lectureColumns+` FROM lectures l WHERE l.id = $1`, id))
		return err
	})
	if err != nil {
		return models.Lecture{}, fmt.Errorf("insert lecture: %w", err)
	}
	return saved, nil
}

// InsertReview stores a review-pass result for a lecture.
func (s *SQLiteStore) InsertReview(ctx context.Context, lectureID string, r models.LectureReview) error {
	_, err := s.c().exec(ctx, `
		INSERT INTO lecture_reviews (id, lecture_id, verdict, issues, model, reviewer, attempt, summary, validation_issues, reader_gaps)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		NewID(), lectureID, r.Verdict, nonNullJSON(r.Issues, "[]"), r.Model, r.Reviewer, r.Attempt, r.Summary,
		nonNullJSON(r.ValidationIssues, "[]"), nonNullJSON(r.ReaderGaps, "[]"))
	if err != nil {
		return fmt.Errorf("insert review: %w", err)
	}
	return nil
}

// LatestLectureForTarget returns the newest lecture of the exact target and
// kind with one of the given statuses, or ErrNotFound.
func (s *SQLiteStore) LatestLectureForTarget(ctx context.Context, kind, conceptID, objectiveKey string, depth int, statuses []string) (models.Lecture, error) {
	l, err := scanLecture(s.c().row(ctx, `
		SELECT `+lectureColumns+` FROM lectures l
		WHERE l.kind = $5 AND l.concept_id = $1 AND l.objective_key = $2 AND l.depth = $3
		  AND l.status IN (SELECT value FROM json_each($4))
		ORDER BY l.version DESC LIMIT 1`, conceptID, objectiveKey, depth, nonNil(statuses), kind))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Lecture{}, ErrNotFound
	}
	return l, err
}

// PublishedLectureCovering returns the newest published lecture of the concept
// at depth whose objectives include all of objectiveIDs (library preferred;
// instant lectures only when requested by userID).
func (s *SQLiteStore) PublishedLectureCovering(ctx context.Context, userID, conceptID string, depth int, objectiveIDs []string) (models.Lecture, error) {
	l, err := scanLecture(s.c().row(ctx, `
		SELECT `+lectureColumns+` FROM lectures l
		WHERE l.concept_id = $1 AND l.depth = $2 AND l.status = 'published'
		  AND NOT EXISTS (
			SELECT 1 FROM json_each($3) want
			WHERE want.value NOT IN (SELECT value FROM json_each(l.objective_ids)))
		  AND (l.kind = 'library' OR l.requested_by = $4)
		ORDER BY (l.kind = 'library') DESC, l.version DESC, l.created_at DESC LIMIT 1`,
		conceptID, depth, nonNil(objectiveIDs), userID))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Lecture{}, ErrNotFound
	}
	return l, err
}

// LectureSummaries returns the @@summary text of the given lectures the user
// may see (library, or instant ones they requested), by lecture id; lectures
// without a summary block are left out.
func (s *SQLiteStore) LectureSummaries(ctx context.Context, userID string, lectureIDs []string) (map[string]string, error) {
	out := map[string]string{}
	if len(lectureIDs) == 0 {
		return out, nil
	}
	type row struct {
		id  string
		raw json.RawMessage
	}
	rows, err := collect(s.c().query(ctx, `
		SELECT l.id, l.content FROM lectures l
		WHERE l.id IN (SELECT value FROM json_each($1)) AND (l.kind = 'library' OR l.requested_by = $2)`, lectureIDs, userID),
		func(r rowScanner) (row, error) {
			var x row
			err := r.Scan(&x.id, rawJSON(&x.raw))
			return x, err
		})
	if err != nil {
		return nil, fmt.Errorf("lecture summaries: %w", err)
	}
	for _, x := range rows {
		var c lecture.Content
		if err := json.Unmarshal(x.raw, &c); err != nil {
			return nil, fmt.Errorf("decode lecture %s: %w", x.id, err)
		}
		if text := c.SummaryText(); text != "" {
			out[x.id] = text
		}
	}
	return out, nil
}

// GetLecture loads a lecture with the user's progress and its reviews.
// Instant lectures are only visible to the user who requested them.
func (s *SQLiteStore) GetLecture(ctx context.Context, userID, lectureID string) (models.Lecture, error) {
	l, err := scanLecture(s.c().row(ctx, `
		SELECT `+lectureColumns+` FROM lectures l
		WHERE l.id = $1 AND (l.kind = 'library' OR l.requested_by = $2)`, lectureID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Lecture{}, ErrNotFound
	}
	if err != nil {
		return models.Lecture{}, fmt.Errorf("get lecture: %w", err)
	}

	var p models.LectureProgress
	err = s.c().row(ctx, `
		SELECT anchor, percent, started_at, last_seen_at, completed_at
		FROM user_lecture_progress WHERE user_id = $1 AND lecture_id = $2`, userID, l.ID).
		Scan(&p.Anchor, &p.Percent, ts(&p.StartedAt), ts(&p.LastSeenAt), nts(&p.CompletedAt))
	if err == nil {
		l.Progress = &p
	} else if !errors.Is(err, sql.ErrNoRows) {
		return models.Lecture{}, fmt.Errorf("get progress: %w", err)
	}

	l.Reviews, err = collect(s.c().query(ctx, `
		SELECT id, verdict, summary, issues, validation_issues, reader_gaps, attempt, model, reviewer, created_at
		FROM lecture_reviews WHERE lecture_id = $1 ORDER BY created_at`, l.ID), func(row rowScanner) (models.LectureReview, error) {
		var r models.LectureReview
		err := row.Scan(&r.ID, &r.Verdict, &r.Summary, rawJSON(&r.Issues), rawJSON(&r.ValidationIssues), rawJSON(&r.ReaderGaps), &r.Attempt, &r.Model, &r.Reviewer, ts(&r.CreatedAt))
		return r, err
	})
	if err != nil {
		return models.Lecture{}, fmt.Errorf("list reviews: %w", err)
	}
	return l, nil
}

// LatestFeedback returns the user's most recent feedback value ("" if none).
func (s *SQLiteStore) LatestFeedback(ctx context.Context, userID string) (string, error) {
	var fb string
	err := s.c().row(ctx, `
		SELECT feedback FROM lecture_feedback WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, userID).Scan(&fb)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return fb, err
}

func nonNullJSON(raw json.RawMessage, fallback string) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage(fallback)
	}
	return raw
}
