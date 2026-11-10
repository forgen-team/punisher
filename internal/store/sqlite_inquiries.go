package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wooo-jin/punisher/internal/models"
)

const inquiryFollowupCountSQL = `(SELECT COUNT(*) FROM inquiry_followups f WHERE f.inquiry_id = inquiries.id)`

const inquiryColumns = `id, question, status, content, concept_ids, proposed_concept_ids,
	model, prompt_version, error, read_at, created_at, updated_at, answered_at, ` + inquiryFollowupCountSQL

// inquirySummaryColumns: 목록용 — 해설 본문은 싣지 않고 제목만(content.title, 목록 행 제목).
const inquirySummaryColumns = `id, question, status, json_object('title', COALESCE(json_extract(content, '$.title'), '')), concept_ids, proposed_concept_ids,
	model, prompt_version, error, read_at, created_at, updated_at, answered_at, ` + inquiryFollowupCountSQL

func scanInquiry(row rowScanner) (models.Inquiry, error) {
	var q models.Inquiry
	err := row.Scan(&q.ID, &q.Question, &q.Status, rawJSON(&q.Content), strs(&q.ConceptIDs), strs(&q.ProposedConceptIDs),
		&q.Model, &q.PromptVersion, &q.Error, nts(&q.ReadAt), ts(&q.CreatedAt), ts(&q.UpdatedAt), nts(&q.AnsweredAt), &q.FollowupCount)
	if q.ConceptIDs == nil {
		q.ConceptIDs = []string{}
	}
	if q.ProposedConceptIDs == nil {
		q.ProposedConceptIDs = []string{}
	}
	return q, err
}

func (s *SQLiteStore) inquiryRow(ctx context.Context, userID, id string) (models.Inquiry, error) {
	q, err := scanInquiry(s.c().row(ctx, `SELECT `+inquiryColumns+` FROM inquiries WHERE user_id = $1 AND id = $2`, userID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return q, ErrNotFound
	}
	return q, err
}

// CreateInquiry inserts a pending inquiry. A pending inquiry with the same
// question created within dedupeWindow is returned instead (double submit);
// created reports whether a new row was made.
func (s *SQLiteStore) CreateInquiry(ctx context.Context, userID, question string, dedupeWindow time.Duration) (q models.Inquiry, created bool, err error) {
	q, err = scanInquiry(s.c().row(ctx, `
		SELECT `+inquiryColumns+` FROM inquiries
		WHERE user_id = $1 AND question = $2 AND status = 'pending' AND created_at > $3
		ORDER BY created_at DESC LIMIT 1`, userID, question, time.Now().Add(-dedupeWindow)))
	if err == nil {
		return q, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return q, false, fmt.Errorf("find pending inquiry: %w", err)
	}
	id := NewID()
	if _, err := s.c().exec(ctx, `INSERT INTO inquiries (id, user_id, question) VALUES ($1, $2, $3)`, id, userID, question); err != nil {
		return q, false, fmt.Errorf("create inquiry: %w", err)
	}
	q, err = s.inquiryRow(ctx, userID, id)
	return q, err == nil, err
}

// CountPendingInquiries counts explanations being written for the user (first answers, regenerations and follow-ups).
func (s *SQLiteStore) CountPendingInquiries(ctx context.Context, userID string) (int, error) {
	var n int
	err := s.c().row(ctx, `
		SELECT (SELECT COUNT(*) FROM inquiries WHERE user_id = $1 AND status = 'pending')
		     + (SELECT COUNT(*) FROM inquiry_followups WHERE user_id = $1 AND status = 'pending')`, userID).Scan(&n)
	return n, err
}

// GetInquiry loads one of the user's inquiries (with content and follow-ups).
func (s *SQLiteStore) GetInquiry(ctx context.Context, userID, id string) (models.Inquiry, error) {
	q, err := s.inquiryRow(ctx, userID, id)
	if err != nil {
		return q, err
	}
	q.Followups, err = s.ListFollowups(ctx, userID, id)
	return q, err
}

// InquiryFilter selects inquiries for ListInquiries.
type InquiryFilter struct {
	// Open: 아직 '다 봤어요' 하지 않은 것만(학습 홈 목록).
	Open bool
	// ConceptID: 이 개념(관련·제안)에 붙은 것만(지도 개념 상세).
	ConceptID string
	Limit     int
}

// ListInquiries returns the user's inquiries, newest first, without content.
func (s *SQLiteStore) ListInquiries(ctx context.Context, userID string, f InquiryFilter) ([]models.Inquiry, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 50
	}
	list, err := collect(s.c().query(ctx, `
		SELECT `+inquirySummaryColumns+` FROM inquiries
		WHERE user_id = $1 AND (NOT $2 OR read_at IS NULL)
		  AND ($3 = '' OR EXISTS (SELECT 1 FROM json_each(concept_ids) WHERE value = $3)
		       OR EXISTS (SELECT 1 FROM json_each(proposed_concept_ids) WHERE value = $3))
		ORDER BY created_at DESC LIMIT $4`, userID, f.Open, f.ConceptID, f.Limit), scanInquiry)
	if err != nil {
		return nil, fmt.Errorf("list inquiries: %w", err)
	}
	return list, nil
}

// SetInquiryPending marks an inquiry as being regenerated; the previous answer
// stays until the new one is saved. started is false when a generation is
// already running (no duplicate generation).
func (s *SQLiteStore) SetInquiryPending(ctx context.Context, userID, id string) (q models.Inquiry, started bool, err error) {
	res, err := s.c().exec(ctx, `
		UPDATE inquiries SET status = 'pending', error = '', updated_at = NOW()
		WHERE user_id = $1 AND id = $2 AND status <> 'pending'`, userID, id)
	if err != nil {
		return q, false, fmt.Errorf("regenerate inquiry: %w", err)
	}
	q, err = s.inquiryRow(ctx, userID, id)
	return q, err == nil && affected(res) > 0, err
}

// InquiryAnswer is a validated explanation to store.
type InquiryAnswer struct {
	Content            json.RawMessage
	ConceptIDs         []string
	ProposedConceptIDs []string
	Model              string
	PromptVersion      string
	Attempts           []AttemptRecord
}

// AnswerInquiry stores a successful explanation (cache: reopening shows it as is).
// Concepts proposed by an earlier version of the answer stay on the thread.
func (s *SQLiteStore) AnswerInquiry(ctx context.Context, userID, id string, a InquiryAnswer) error {
	log, err := json.Marshal(attemptsOrEmpty(a.Attempts))
	if err != nil {
		return err
	}
	_, err = s.c().exec(ctx, `
		UPDATE inquiries SET status = 'answered', content = $3, concept_ids = $4,
			proposed_concept_ids = (SELECT json_group_array(value) FROM (
				SELECT value FROM json_each(inquiries.proposed_concept_ids) UNION SELECT value FROM json_each($5))),
			model = $6, prompt_version = $7, attempt_log = $8, error = '', answered_at = NOW(), updated_at = NOW()
		WHERE user_id = $1 AND id = $2`, userID, id, a.Content, nonNil(a.ConceptIDs), nonNil(a.ProposedConceptIDs), a.Model, a.PromptVersion, string(log))
	if err != nil {
		return fmt.Errorf("answer inquiry: %w", err)
	}
	return nil
}

// FailInquiry records a failed generation (never stored as answered). A
// previous answer and its model are kept so a failed regeneration loses nothing.
func (s *SQLiteStore) FailInquiry(ctx context.Context, userID, id, model, errMsg string, attempts []AttemptRecord) error {
	log, err := json.Marshal(attemptsOrEmpty(attempts))
	if err != nil {
		return err
	}
	_, err = s.c().exec(ctx, `
		UPDATE inquiries SET status = CASE WHEN answered_at IS NULL THEN 'failed' ELSE 'answered' END,
			model = CASE WHEN answered_at IS NULL THEN $3 ELSE model END,
			error = $4, attempt_log = $5, updated_at = NOW()
		WHERE user_id = $1 AND id = $2`, userID, id, model, truncateText(errMsg, 2000), string(log))
	if err != nil {
		return fmt.Errorf("fail inquiry: %w", err)
	}
	return nil
}

// SetInquiryRead marks an inquiry as read ('다 봤어요', drops off the home list) or restores it.
func (s *SQLiteStore) SetInquiryRead(ctx context.Context, userID, id string, read bool) error {
	res, err := s.c().exec(ctx, `
		UPDATE inquiries SET read_at = CASE WHEN $3 THEN COALESCE(read_at, NOW()) END, updated_at = NOW()
		WHERE user_id = $1 AND id = $2`, userID, id, read)
	if err != nil {
		return fmt.Errorf("read inquiry: %w", err)
	}
	if affected(res) == 0 {
		return ErrNotFound
	}
	return nil
}

// SweepStaleInquiries ends generations that stopped without finishing (server
// restart, crash): pending rows older than olderThan become failed, or go back
// to answered when a previous answer exists. Returns the rows changed.
func (s *SQLiteStore) SweepStaleInquiries(ctx context.Context, olderThan time.Duration) (int64, error) {
	const msg = "generation interrupted"
	cutoff := time.Now().Add(-olderThan)
	a, err := s.c().exec(ctx, `
		UPDATE inquiries SET status = CASE WHEN answered_at IS NULL THEN 'failed' ELSE 'answered' END, error = $2, updated_at = NOW()
		WHERE status = 'pending' AND updated_at <= $1`, cutoff, msg)
	if err != nil {
		return 0, fmt.Errorf("sweep inquiries: %w", err)
	}
	b, err := s.c().exec(ctx, `
		UPDATE inquiry_followups SET status = 'failed', error = $2, updated_at = NOW()
		WHERE status = 'pending' AND updated_at <= $1`, cutoff, msg)
	if err != nil {
		return 0, fmt.Errorf("sweep inquiry followups: %w", err)
	}
	return affected(a) + affected(b), nil
}

// InquiryConcepts returns name/area/state of the given concepts the user can see,
// in the order of ids (unknown ids are skipped).
func (s *SQLiteStore) InquiryConcepts(ctx context.Context, userID string, ids []string) ([]models.ConceptLink, error) {
	if len(ids) == 0 {
		return []models.ConceptLink{}, nil
	}
	list, err := collect(s.c().query(ctx, `
		SELECT c.id, c.name, c.area, CASE WHEN c.status = 'proposed' THEN 'proposed' ELSE 'related' END,
			COALESCE(st.state, 'not_started')
		FROM json_each($2) want
		JOIN concepts c ON c.id = want.value
		LEFT JOIN user_concept_states st ON st.concept_id = c.id AND st.user_id = $1
		WHERE c.status <> 'retired' AND `+visibleTo("c", "$1")+`
		ORDER BY want.key`, userID, ids), func(row rowScanner) (models.ConceptLink, error) {
		var l models.ConceptLink
		err := row.Scan(&l.ID, &l.Name, &l.Area, &l.Kind, &l.UserState)
		return l, err
	})
	if err != nil {
		return nil, fmt.Errorf("inquiry concepts: %w", err)
	}
	return list, nil
}

func attemptsOrEmpty(a []AttemptRecord) []AttemptRecord {
	if a == nil {
		return []AttemptRecord{}
	}
	return a
}

const followupColumns = `id, question, quote, status, section, error, created_at, answered_at`

func scanFollowup(row rowScanner) (models.InquiryFollowup, error) {
	var f models.InquiryFollowup
	err := row.Scan(&f.ID, &f.Question, &f.Quote, &f.Status, rawJSON(&f.Section), &f.Error, ts(&f.CreatedAt), nts(&f.AnsweredAt))
	return f, err
}

// ListFollowups returns a thread's follow-ups in order.
func (s *SQLiteStore) ListFollowups(ctx context.Context, userID, inquiryID string) ([]models.InquiryFollowup, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT `+followupColumns+` FROM inquiry_followups
		WHERE user_id = $1 AND inquiry_id = $2 ORDER BY created_at, id`, userID, inquiryID), scanFollowup)
	if err != nil {
		return nil, fmt.Errorf("list followups: %w", err)
	}
	return list, nil
}

// CreateFollowup adds a pending follow-up to a thread and puts the thread back on the home list.
func (s *SQLiteStore) CreateFollowup(ctx context.Context, userID, inquiryID, question, quote string) (models.InquiryFollowup, error) {
	var f models.InquiryFollowup
	err := s.tx(ctx, func(c conn) error {
		var err error
		f, err = scanFollowup(c.row(ctx, `
			INSERT INTO inquiry_followups (id, inquiry_id, user_id, question, quote)
			VALUES ($1, $2, $3, $4, $5) RETURNING `+followupColumns, NewID(), inquiryID, userID, question, quote))
		if err != nil {
			return fmt.Errorf("create followup: %w", err)
		}
		_, err = c.exec(ctx, `UPDATE inquiries SET read_at = NULL, updated_at = NOW() WHERE user_id = $1 AND id = $2`, userID, inquiryID)
		return err
	})
	return f, err
}

// SetFollowupPending restarts a failed follow-up (다시 시도).
func (s *SQLiteStore) SetFollowupPending(ctx context.Context, userID, id string) (models.InquiryFollowup, error) {
	f, err := scanFollowup(s.c().row(ctx, `
		UPDATE inquiry_followups SET status = 'pending', error = '', updated_at = NOW()
		WHERE user_id = $1 AND id = $2 AND status = 'failed' RETURNING `+followupColumns, userID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return f, ErrNotFound
	}
	return f, err
}

// FollowupAnswer is a validated follow-up answer; its concepts are merged into the thread.
type FollowupAnswer struct {
	Section            json.RawMessage
	ConceptIDs         []string
	ProposedConceptIDs []string
	Model              string
}

// AnswerFollowup stores a follow-up answer and sets the thread's concept lists
// (merged by the caller, so the thread keeps one card per node).
func (s *SQLiteStore) AnswerFollowup(ctx context.Context, userID, inquiryID, id string, a FollowupAnswer) error {
	return s.tx(ctx, func(c conn) error {
		if _, err := c.exec(ctx, `
			UPDATE inquiry_followups SET status = 'answered', section = $3, model = $4, error = '', answered_at = NOW(), updated_at = NOW()
			WHERE user_id = $1 AND id = $2`, userID, id, a.Section, a.Model); err != nil {
			return fmt.Errorf("answer followup: %w", err)
		}
		if _, err := c.exec(ctx, `
			UPDATE inquiries SET concept_ids = $3, proposed_concept_ids = $4, updated_at = NOW()
			WHERE user_id = $1 AND id = $2`, userID, inquiryID, nonNil(a.ConceptIDs), nonNil(a.ProposedConceptIDs)); err != nil {
			return fmt.Errorf("merge followup concepts: %w", err)
		}
		return nil
	})
}

// FailFollowup records a failed follow-up answer (never stored as answered).
func (s *SQLiteStore) FailFollowup(ctx context.Context, userID, id, model, errMsg string) error {
	_, err := s.c().exec(ctx, `
		UPDATE inquiry_followups SET status = 'failed', model = $3, error = $4, updated_at = NOW()
		WHERE user_id = $1 AND id = $2`, userID, id, model, truncateText(errMsg, 2000))
	if err != nil {
		return fmt.Errorf("fail followup: %w", err)
	}
	return nil
}
