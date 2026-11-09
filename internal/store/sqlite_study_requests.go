package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/wooo-jin/punisher/internal/models"
)

const studyRequestColumns = `id, user_id, source, source_ref, query, context,
	resolved_concept_ids, status, idempotency_key, COALESCE(starting_concept_id, ''),
	resolve_attempts, resolve_error, resolved_at, created_at, updated_at`

func scanStudyRequest(row rowScanner) (models.StudyRequest, error) {
	var r models.StudyRequest
	err := row.Scan(&r.ID, &r.UserID, &r.Source, &r.SourceRef, &r.Query, rawJSON(&r.Context),
		strs(&r.ResolvedConceptIDs), &r.Status, &r.IdempotencyKey, &r.StartingConceptID,
		&r.ResolveAttempts, &r.ResolveError, nts(&r.ResolvedAt), ts(&r.CreatedAt), ts(&r.UpdatedAt))
	return r, err
}

// CreateStudyRequest inserts the request unless (user_id, idempotency_key)
// already exists, in which case the existing row is returned with created=false.
func (s *SQLiteStore) CreateStudyRequest(ctx context.Context, req models.StudyRequest) (models.StudyRequest, bool, error) {
	created, err := scanStudyRequest(s.c().row(ctx, `
		INSERT INTO study_requests (id, user_id, source, source_ref, query, context,
		                            resolved_concept_ids, status, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, idempotency_key) DO NOTHING
		RETURNING `+studyRequestColumns,
		NewID(), req.UserID, req.Source, req.SourceRef, req.Query, nonNullJSON(req.Context, "{}"),
		nonNil(req.ResolvedConceptIDs), req.Status, req.IdempotencyKey))
	if err == nil {
		return created, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return models.StudyRequest{}, false, fmt.Errorf("insert study request: %w", err)
	}

	existing, err := scanStudyRequest(s.c().row(ctx, `
		SELECT `+studyRequestColumns+`
		FROM study_requests WHERE user_id = $1 AND idempotency_key = $2`,
		req.UserID, req.IdempotencyKey))
	if err != nil {
		return models.StudyRequest{}, false, fmt.Errorf("load existing study request: %w", err)
	}
	return existing, false, nil
}

// ListStudyRequests returns the user's requests ("" status = all), newest first,
// with Learned computed from the resolved concepts' states.
func (s *SQLiteStore) ListStudyRequests(ctx context.Context, userID, status string, limit int) ([]models.StudyRequest, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT `+studyRequestColumns+` FROM study_requests
		WHERE user_id = $1 AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC LIMIT $3`, userID, status, limit), scanStudyRequest)
	if err != nil {
		return nil, fmt.Errorf("list study requests: %w", err)
	}
	for i := range list {
		if list[i].Learned, err = s.allDone(ctx, userID, list[i].ResolvedConceptIDs); err != nil {
			return nil, err
		}
	}
	return list, nil
}

// GetStudyRequest loads one request with Learned computed.
func (s *SQLiteStore) GetStudyRequest(ctx context.Context, userID, id string) (models.StudyRequest, error) {
	r, err := scanStudyRequest(s.c().row(ctx,
		`SELECT `+studyRequestColumns+` FROM study_requests WHERE user_id = $1 AND id = $2`, userID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	r.Learned, err = s.allDone(ctx, userID, r.ResolvedConceptIDs)
	return r, err
}

func (s *SQLiteStore) allDone(ctx context.Context, userID string, conceptIDs []string) (bool, error) {
	if len(conceptIDs) == 0 {
		return false, nil
	}
	var done int
	err := s.c().row(ctx, `
		SELECT COUNT(*) FROM user_concept_states
		WHERE user_id = $1 AND concept_id IN (SELECT value FROM json_each($2)) AND state IN ('completed', 'known')`,
		userID, conceptIDs).Scan(&done)
	return done == len(conceptIDs), err
}

// UnresolvedStudyRequests returns requests the worker should (re)try.
func (s *SQLiteStore) UnresolvedStudyRequests(ctx context.Context, userID string, maxAttempts, limit int) ([]models.StudyRequest, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT `+studyRequestColumns+` FROM study_requests
		WHERE user_id = $1 AND status = 'unresolved' AND resolve_attempts < $2
		ORDER BY created_at LIMIT $3`, userID, maxAttempts, limit), scanStudyRequest)
	if err != nil {
		return nil, fmt.Errorf("unresolved study requests: %w", err)
	}
	return list, nil
}

// RecordResolution stores a resolution attempt. Empty conceptIDs with errMsg
// keeps the request unresolved and counts the attempt.
func (s *SQLiteStore) RecordResolution(ctx context.Context, id string, conceptIDs []string, startingConceptID, errMsg string) error {
	var starting any
	if startingConceptID != "" {
		starting = startingConceptID
	}
	_, err := s.c().exec(ctx, `
		UPDATE study_requests SET
			resolved_concept_ids = CASE WHEN json_array_length($2) > 0 THEN $2 ELSE resolved_concept_ids END,
			starting_concept_id = COALESCE($3, starting_concept_id),
			status = CASE WHEN json_array_length($2) > 0 AND status = 'unresolved' THEN 'resolved' ELSE status END,
			resolved_at = CASE WHEN json_array_length($2) > 0 THEN NOW() ELSE resolved_at END,
			resolve_attempts = resolve_attempts + 1, resolve_error = $4, updated_at = NOW()
		WHERE id = $1`, id, nonNil(conceptIDs), starting, truncateText(errMsg, 1000))
	return err
}
