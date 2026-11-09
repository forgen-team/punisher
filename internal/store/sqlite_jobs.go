package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wooo-jin/punisher/internal/models"
)

// ErrJobActive means another pending/running job already targets the same
// (objective_key, depth) — concurrent generation is blocked.
var ErrJobActive = errors.New("a generation job for this target is already active")

const jobColumns = `id, idempotency_key, kind, COALESCE(user_id, ''), COALESCE(concept_id, ''),
	objective_key, COALESCE(depth, 0), status, reason, error, attempts, COALESCE(lecture_id, ''),
	model, input_tokens, output_tokens, input, started_at, finished_at, created_at`

func scanJob(row rowScanner) (models.GenerationJob, error) {
	var j models.GenerationJob
	err := row.Scan(&j.ID, &j.IdempotencyKey, &j.Kind, &j.UserID, &j.ConceptID, &j.ObjectiveKey, &j.Depth,
		&j.Status, &j.Reason, &j.Error, &j.Attempts, &j.LectureID, &j.Model, &j.InputTokens, &j.OutputTokens,
		rawJSON(&j.Input), nts(&j.StartedAt), nts(&j.FinishedAt), ts(&j.CreatedAt))
	return j, err
}

// NewJob is the insert payload for a generation job.
type NewJob struct {
	IdempotencyKey string
	Kind           string
	UserID         string
	ConceptID      string
	ObjectiveKey   string
	Depth          int
	Reason         string
	Input          json.RawMessage
}

// CreateJob inserts a pending job. When the idempotency key already exists the
// existing job is returned with created=false. When another active job holds
// the same target, ErrJobActive is returned together with that job.
func (s *SQLiteStore) CreateJob(ctx context.Context, j NewJob) (models.GenerationJob, bool, error) {
	var conceptID, depth any
	if j.ConceptID != "" {
		conceptID = j.ConceptID
	}
	if j.Depth > 0 {
		depth = j.Depth
	}
	job, err := scanJob(s.c().row(ctx, `
		INSERT INTO generation_jobs (id, idempotency_key, kind, user_id, concept_id, objective_key, depth, reason, input)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING `+jobColumns,
		NewID(), j.IdempotencyKey, j.Kind, j.UserID, conceptID, j.ObjectiveKey, depth, j.Reason, nonNullJSON(j.Input, "{}")))
	if err == nil {
		return job, true, nil
	}
	if isUniqueViolation(err) {
		// The only other unique constraint is the partial index
		// generation_jobs_one_active_target (one active job per target).
		active, aerr := s.ActiveJobForTarget(ctx, j.ObjectiveKey, j.Depth)
		if aerr != nil {
			return models.GenerationJob{}, false, ErrJobActive
		}
		return active, false, ErrJobActive
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return models.GenerationJob{}, false, fmt.Errorf("insert job: %w", err)
	}
	existing, err := scanJob(s.c().row(ctx,
		`SELECT `+jobColumns+` FROM generation_jobs WHERE idempotency_key = $1`, j.IdempotencyKey))
	if err != nil {
		return models.GenerationJob{}, false, fmt.Errorf("load existing job: %w", err)
	}
	return existing, false, nil
}

// ActiveJobForTarget returns the pending/running lecture job for a target.
func (s *SQLiteStore) ActiveJobForTarget(ctx context.Context, objectiveKey string, depth int) (models.GenerationJob, error) {
	j, err := scanJob(s.c().row(ctx, `
		SELECT `+jobColumns+` FROM generation_jobs
		WHERE objective_key = $1 AND depth = $2 AND status IN ('pending', 'running') AND kind IN ('library', 'instant')
		LIMIT 1`, objectiveKey, depth))
	if errors.Is(err, sql.ErrNoRows) {
		return models.GenerationJob{}, ErrNotFound
	}
	return j, err
}

// GetJob loads a job owned by userID.
func (s *SQLiteStore) GetJob(ctx context.Context, userID, jobID string) (models.GenerationJob, error) {
	j, err := scanJob(s.c().row(ctx,
		`SELECT `+jobColumns+` FROM generation_jobs WHERE id = $1 AND user_id = $2`, jobID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return models.GenerationJob{}, ErrNotFound
	}
	return j, err
}

// StartJob marks a job running and bumps attempts.
func (s *SQLiteStore) StartJob(ctx context.Context, jobID string) error {
	_, err := s.c().exec(ctx, `
		UPDATE generation_jobs SET status = 'running', attempts = attempts + 1,
			started_at = COALESCE(started_at, NOW()), updated_at = NOW()
		WHERE id = $1`, jobID)
	return err
}

// JobOutcome is the terminal state of a job.
type JobOutcome struct {
	Status       string // succeeded | failed | skipped
	Reason       string
	Error        string
	LectureID    string
	Model        string
	InputTokens  int
	OutputTokens int
	// Attempts: per-attempt record of multi-attempt jobs (nil = keep the column as is).
	Attempts []AttemptRecord
}

// AttemptRecord is one generation attempt of a job (generation_jobs.attempt_log).
type AttemptRecord struct {
	Attempt int `json:"attempt"`
	// Result: ok | parse_error | invalid (schema issues) | review_failed
	Result  string `json:"result"`
	Message string `json:"message,omitempty"`
}

// FinishJob records the terminal state. Failed jobs never carry a lecture.
func (s *SQLiteStore) FinishJob(ctx context.Context, jobID string, o JobOutcome) error {
	var lectureID any
	if o.LectureID != "" && o.Status == "succeeded" {
		lectureID = o.LectureID
	}
	var attemptLog any // NULL keeps the column as is
	if o.Attempts != nil {
		raw, err := json.Marshal(o.Attempts)
		if err != nil {
			return err
		}
		attemptLog = string(raw)
	}
	_, err := s.c().exec(ctx, `
		UPDATE generation_jobs SET status = $2, reason = CASE WHEN $3 = '' THEN reason ELSE $3 END,
			error = $4, lecture_id = $5, model = $6, input_tokens = input_tokens + $7,
			output_tokens = output_tokens + $8, finished_at = NOW(), updated_at = NOW(),
			attempt_log = COALESCE($9, attempt_log),
			attempts = CASE WHEN $9 IS NULL THEN attempts ELSE json_array_length($9) END
		WHERE id = $1`, jobID, o.Status, o.Reason, truncateText(o.Error, 2000), lectureID, o.Model,
		o.InputTokens, o.OutputTokens, attemptLog)
	return err
}

// RecordUsage stores a finished accounting-only job (question, gloss, resolve)
// so daily token budgets include every LLM call and failures are visible.
func (s *SQLiteStore) RecordUsage(ctx context.Context, userID, kind, conceptID string, o JobOutcome) error {
	var concept any
	if conceptID != "" {
		concept = conceptID
	}
	_, err := s.c().exec(ctx, `
		INSERT INTO generation_jobs (id, idempotency_key, kind, user_id, concept_id, status, reason, error,
			model, input_tokens, output_tokens, attempts, started_at, finished_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9, $10, $11, 1, NOW(), NOW())`,
		NewID(), kind+":"+RandomToken(), kind, userID, concept, o.Status, o.Reason, truncateText(o.Error, 2000),
		o.Model, o.InputTokens, o.OutputTokens)
	return err
}

// UsageSince returns lecture-job count (library+instant, excluding skipped)
// and total tokens across all kinds for a user since t.
func (s *SQLiteStore) UsageSince(ctx context.Context, userID string, since time.Time) (lectures int, tokens int, err error) {
	err = s.c().row(ctx, `
		SELECT COUNT(*) FILTER (WHERE kind IN ('library', 'instant') AND status <> 'skipped'),
		       COALESCE(SUM(input_tokens + output_tokens), 0)
		FROM generation_jobs WHERE user_id = $1 AND created_at >= $2`, userID, since).Scan(&lectures, &tokens)
	return lectures, tokens, err
}

// FailStaleJobs marks jobs stuck in pending/running (e.g. after a restart) as
// failed so they stop blocking their target.
func (s *SQLiteStore) FailStaleJobs(ctx context.Context, olderThan time.Duration) (int64, error) {
	res, err := s.c().exec(ctx, `
		UPDATE generation_jobs SET status = 'failed', error = 'stale: no progress (server restart or timeout)',
			finished_at = NOW(), updated_at = NOW()
		WHERE status IN ('pending', 'running') AND updated_at < $1`,
		time.Now().Add(-olderThan))
	if err != nil {
		return 0, err
	}
	return affected(res), nil
}

func truncateText(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// RandomToken returns a 128-bit random hex string for one-off keys.
func RandomToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	return hex.EncodeToString(b[:])
}
