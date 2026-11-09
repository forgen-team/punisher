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

// studyRequestLearnedSQL: 요청한 주제 개념(resolved_concept_ids 의 첫 번째)을 다 배웠는가 — 개념이 completed/known 이거나,
// 그 개념의 강의를 하나라도 끝까지 봤으면(완료 처리) 완료로 친다. 선행 개념까지 포함한 '전체 완료'는 사실상
// 오지 않아(선행 개념은 요청한 주제가 아님) 다 본 요청이 목록에 계속 남았다(witems cf8a322).
// 강의 하나로는 개념의 모든 목표(objective)가 끝나지 않아 개념 상태가 in_progress 로 남는 경우도 강의 완료로 잡는다.
const studyRequestLearnedSQL = `(r.status = 'done' OR (json_array_length(r.resolved_concept_ids) > 0 AND (
	EXISTS (SELECT 1 FROM user_concept_states s
	        WHERE s.user_id = r.user_id AND s.concept_id = json_extract(r.resolved_concept_ids, '$[0]')
	          AND s.state IN ('completed', 'known'))
	OR EXISTS (SELECT 1 FROM user_lecture_progress p JOIN lectures l ON l.id = p.lecture_id
	           WHERE p.user_id = r.user_id AND p.completed_at IS NOT NULL
	             AND l.concept_id = json_extract(r.resolved_concept_ids, '$[0]')))))`

// studyRequestColumnsR is studyRequestColumns qualified with the "r" alias.
const studyRequestColumnsR = `r.id, r.user_id, r.source, r.source_ref, r.query, r.context,
	r.resolved_concept_ids, r.status, r.idempotency_key, COALESCE(r.starting_concept_id, ''),
	r.resolve_attempts, r.resolve_error, r.resolved_at, r.created_at, r.updated_at`

func scanStudyRequestWithLearned(row rowScanner) (models.StudyRequest, error) {
	var r models.StudyRequest
	err := row.Scan(&r.ID, &r.UserID, &r.Source, &r.SourceRef, &r.Query, rawJSON(&r.Context),
		strs(&r.ResolvedConceptIDs), &r.Status, &r.IdempotencyKey, &r.StartingConceptID,
		&r.ResolveAttempts, &r.ResolveError, nts(&r.ResolvedAt), ts(&r.CreatedAt), ts(&r.UpdatedAt), &r.Learned)
	return r, err
}

// ListStudyRequests returns the user's requests ("" status = all), newest first,
// with Learned computed for the requested topic concept (one query, no per-row lookups).
func (s *SQLiteStore) ListStudyRequests(ctx context.Context, userID, status string, limit int) ([]models.StudyRequest, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT `+studyRequestColumnsR+`, `+studyRequestLearnedSQL+` FROM study_requests r
		WHERE r.user_id = $1 AND ($2 = '' OR r.status = $2)
		ORDER BY r.created_at DESC LIMIT $3`, userID, status, limit), scanStudyRequestWithLearned)
	if err != nil {
		return nil, fmt.Errorf("list study requests: %w", err)
	}
	return list, nil
}

// GetStudyRequest loads one request with Learned computed.
func (s *SQLiteStore) GetStudyRequest(ctx context.Context, userID, id string) (models.StudyRequest, error) {
	r, err := scanStudyRequestWithLearned(s.c().row(ctx,
		`SELECT `+studyRequestColumnsR+`, `+studyRequestLearnedSQL+` FROM study_requests r WHERE r.user_id = $1 AND r.id = $2`, userID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

// SetStudyRequestDismissed 는 학습 요청을 '목록에서 빼기'(dismissed) 하거나 되돌린다(해석 여부에 따라 resolved/unresolved).
func (s *SQLiteStore) SetStudyRequestDismissed(ctx context.Context, userID, id string, dismissed bool) error {
	res, err := s.c().exec(ctx, `
		UPDATE study_requests SET
			status = CASE WHEN $3 THEN 'dismissed'
			              WHEN status <> 'dismissed' THEN status
			              WHEN json_array_length(resolved_concept_ids) > 0 THEN 'resolved'
			              ELSE 'unresolved' END,
			updated_at = NOW()
		WHERE user_id = $1 AND id = $2`, userID, id, dismissed)
	if err != nil {
		return fmt.Errorf("dismiss study request: %w", err)
	}
	if affected(res) == 0 {
		return ErrNotFound
	}
	return nil
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
