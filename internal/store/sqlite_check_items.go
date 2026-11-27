package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
)

// CheckItem is a stored question (check_items).
type CheckItem struct {
	ID          string
	ConceptID   string
	ObjectiveID string // "" = integrative
	LectureID   string
	Purpose     string
	Format      string
	Prompt      string
	Material    json.RawMessage
	Rubric      check.Rubric
	Status      string
	// Language: ko|en — the lecture's for a lecture set, the creator's locale otherwise (i18n §1).
	Language string
}

// CheckItemKey is the cache key of a lecture's item set (design §4.1):
// the default prompt is shared (PromptHash "", CreatedFor ""), an edited
// check.items block makes the set the user's own. Language is the items'
// language: the lecture's for its set, the user's for review variants (i18n §9);
// "" = the lecture's.
type CheckItemKey struct {
	LectureID  string
	Purpose    string
	PromptHash string
	CreatedFor string
	Language   string
}

// JobKey is the generation_jobs idempotency key of the set (ko keeps the
// pre-i18n key, other languages get a "|<lang>" suffix).
func (k CheckItemKey) JobKey() string {
	return "check_items:" + k.LectureID + ":" + k.Purpose + ":" + k.PromptHash + ":" + k.CreatedFor + learning.LanguageSuffix(k.Language)
}

// keyLanguageSQL is the item language of a key: the given one, else the lecture's, else ko.
func keyLanguageSQL(langParam, lectureParam string) string {
	return `COALESCE(NULLIF(` + langParam + `, ''), (SELECT kl.language FROM lectures kl WHERE kl.id = ` + lectureParam + `), 'ko')`
}

const checkItemColumns = `ci.id, ci.concept_id, COALESCE(ci.objective_id, ''), COALESCE(ci.lecture_id, ''),
	ci.purpose, ci.format, ci.prompt, ci.material, ci.rubric, ci.status, ci.language`

func scanCheckItem(row rowScanner) (CheckItem, error) {
	var it CheckItem
	var rubric json.RawMessage
	err := row.Scan(&it.ID, &it.ConceptID, &it.ObjectiveID, &it.LectureID, &it.Purpose, &it.Format,
		&it.Prompt, rawJSON(&it.Material), rawJSON(&rubric), &it.Status, &it.Language)
	if err != nil {
		return it, err
	}
	if len(rubric) > 0 {
		if err := json.Unmarshal(rubric, &it.Rubric); err != nil {
			return it, fmt.Errorf("decode check item rubric: %w", err)
		}
	}
	return it, nil
}

// checkItemVisible is the visibility rule of an item for user $1 (design §3.2).
var checkItemVisible = `EXISTS (SELECT 1 FROM concepts c WHERE c.id = ci.concept_id AND ` + visibleTo("c", "$1") + `)
	AND (ci.lecture_id IS NULL OR EXISTS (SELECT 1 FROM lectures l WHERE l.id = ci.lecture_id AND (l.kind = 'library' OR l.requested_by = $1)))
	AND (ci.prompt_hash = '' OR ci.created_for = $1)`

// ActiveCheckItems returns the active items of a lecture set visible to the user, in the key's language.
// Both purposes come back when key.Purpose is "check" (review items are the set's variants).
func (s *SQLiteStore) ActiveCheckItems(ctx context.Context, userID string, key CheckItemKey) ([]CheckItem, error) {
	list, err := collect(s.c().query(ctx, `SELECT `+checkItemColumns+` FROM check_items ci
		WHERE ci.lecture_id = $2 AND ci.prompt_hash = $3 AND COALESCE(ci.created_for, '') = $4
		  AND ci.language = `+keyLanguageSQL("$5", "$2")+`
		  AND ci.status = 'active' AND `+checkItemVisible+`
		ORDER BY ci.created_at, ci.rowid`, userID, key.LectureID, key.PromptHash, key.CreatedFor, key.Language), scanCheckItem)
	if err != nil {
		return nil, fmt.Errorf("active check items: %w", err)
	}
	return list, nil
}

// CheckItemsByID loads items by id (retired ones too: a session keeps its items), in the order of ids.
func (s *SQLiteStore) CheckItemsByID(ctx context.Context, userID string, ids []string) ([]CheckItem, error) {
	return checkItemsByID(ctx, s.c(), userID, ids)
}

func checkItemsByID(ctx context.Context, c conn, userID string, ids []string) ([]CheckItem, error) {
	if len(ids) == 0 {
		return []CheckItem{}, nil
	}
	list, err := collect(c.query(ctx, `SELECT `+checkItemColumns+` FROM check_items ci
		JOIN json_each($2) w ON w.value = ci.id
		WHERE `+checkItemVisible+`
		ORDER BY w.key`, userID, ids), scanCheckItem)
	if err != nil {
		return nil, fmt.Errorf("check items: %w", err)
	}
	return list, nil
}

// SaveCheckItems stores a validated set in one transaction: the key's previous
// active items (same language) are retired, the new ones record their
// objective's current text/depth, and review items point to their objective's
// check item (variant_of).
func (s *SQLiteStore) SaveCheckItems(ctx context.Context, key CheckItemKey, conceptID, jobID, model, promptVersion string, items []check.Item) ([]CheckItem, error) {
	out := make([]CheckItem, 0, len(items))
	err := s.tx(ctx, func(c conn) error {
		if _, err := c.exec(ctx, `UPDATE check_items SET status = 'retired'
			WHERE lecture_id = $1 AND prompt_hash = $2 AND COALESCE(created_for, '') = $3 AND status = 'active'
			  AND language = `+keyLanguageSQL("$4", "$1"),
			key.LectureID, key.PromptHash, key.CreatedFor, key.Language); err != nil {
			return fmt.Errorf("retire previous check items: %w", err)
		}
		checkOf := map[string]string{}
		for _, purpose := range []string{check.PurposeCheck, check.PurposeReview} {
			for _, it := range items {
				if it.Purpose != purpose {
					continue
				}
				saved, err := insertCheckItem(ctx, c, key, conceptID, jobID, model, promptVersion, it, checkOf[it.ObjectiveID])
				if err != nil {
					return err
				}
				if purpose == check.PurposeCheck && it.ObjectiveID != "" {
					checkOf[it.ObjectiveID] = saved.ID
				}
				out = append(out, saved)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func insertCheckItem(ctx context.Context, c conn, key CheckItemKey, conceptID, jobID, model, version string, it check.Item, variantOf string) (CheckItem, error) {
	rubric, err := json.Marshal(it.Rubric)
	if err != nil {
		return CheckItem{}, err
	}
	id := NewID()
	if _, err := c.exec(ctx, `
		INSERT INTO check_items (id, concept_id, objective_id, lecture_id, purpose, format, prompt, material, rubric,
			variant_of, objective_text, objective_depth, prompt_hash, created_for, generation_job_id, model, prompt_version,
			language, objective_text_en)
		SELECT $15, $1, NULLIF($2, ''), $3, $4, $5, $6, $7, $8, NULLIF($9, ''),
			COALESCE((SELECT text FROM objectives WHERE id = $2), ''), COALESCE((SELECT depth FROM objectives WHERE id = $2), 0),
			$10, NULLIF($11, ''), NULLIF($12, ''), $13, $14, `+keyLanguageSQL("$16", "$3")+`,
			COALESCE((SELECT text_en FROM objectives WHERE id = $2), '')`,
		conceptID, it.ObjectiveID, key.LectureID, it.Purpose, it.Format, it.Prompt, nonNullJSON(it.Material, "{}"), string(rubric),
		variantOf, key.PromptHash, key.CreatedFor, jobID, model, version, id, key.Language); err != nil {
		return CheckItem{}, fmt.Errorf("insert check item: %w", err)
	}
	saved, err := scanCheckItem(c.row(ctx, `SELECT `+checkItemColumns+` FROM check_items ci WHERE ci.id = $1`, id))
	if err != nil {
		return CheckItem{}, fmt.Errorf("load check item: %w", err)
	}
	return saved, nil
}

// StartCheckItemsJob starts the one generation of a set (design §3.2): a
// running job for the key is returned with started=false; otherwise a running
// job is inserted — the first with the plain key, a regeneration (failed job,
// retired items) with "#n" — subject to the user's pending limit
// (ErrCheckBusy). witems serialises this under advisory locks on the key and
// the user; here the transaction alone does, on the single connection.
// ActiveCheckItemsJob is userID's pending/running item-set generation for key
// (ErrNotFound when none runs).
func (s *SQLiteStore) ActiveCheckItemsJob(ctx context.Context, userID string, key CheckItemKey) (models.GenerationJob, error) {
	return s.activeItemsJob(ctx, userID, key.JobKey())
}

// SessionItemsJobKey is the objective_key of a session's own item generation
// (diagnostic, synthesis): one generation per session, found by the session.
func SessionItemsJobKey(sessionID string) string { return "session:" + sessionID }

// StartSessionItemsJob records a running item generation of one session
// (kind check_items, objective_key SessionItemsJobKey) so it can be cancelled.
func (s *SQLiteStore) StartSessionItemsJob(ctx context.Context, userID, conceptID, sessionID string) (models.GenerationJob, error) {
	var concept any
	if conceptID != "" {
		concept = conceptID
	}
	key := SessionItemsJobKey(sessionID)
	return scanJob(s.c().row(ctx, `
		INSERT INTO generation_jobs (id, idempotency_key, kind, user_id, concept_id, objective_key, status, attempts, started_at)
		VALUES ($1, $2, 'check_items', NULLIF($3, ''), (SELECT id FROM concepts WHERE id = $4), $5, 'running', 1, NOW())
		RETURNING `+jobColumns, NewID(), "check_items:"+key+":"+RandomToken(), userID, concept, key))
}

// ActiveSessionItemsJob is the running item generation of a session (ErrNotFound when none runs).
func (s *SQLiteStore) ActiveSessionItemsJob(ctx context.Context, userID, sessionID string) (models.GenerationJob, error) {
	return s.activeItemsJob(ctx, userID, SessionItemsJobKey(sessionID))
}

func (s *SQLiteStore) activeItemsJob(ctx context.Context, userID, objectiveKey string) (models.GenerationJob, error) {
	job, err := scanJob(s.c().row(ctx, `SELECT `+jobColumns+` FROM generation_jobs
		WHERE kind = 'check_items' AND objective_key = $1 AND user_id = $2 AND status IN ('pending', 'running')
		ORDER BY created_at DESC LIMIT 1`, objectiveKey, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return models.GenerationJob{}, ErrNotFound
	}
	return job, err
}

func (s *SQLiteStore) StartCheckItemsJob(ctx context.Context, userID, conceptID string, key CheckItemKey, maxPending int) (models.GenerationJob, bool, error) {
	var job models.GenerationJob
	started := false
	err := s.tx(ctx, func(c conn) error {
		base := key.JobKey()
		running, err := scanJob(c.row(ctx, `SELECT `+jobColumns+` FROM generation_jobs
			WHERE kind = 'check_items' AND objective_key = $1 AND status IN ('pending', 'running')
			ORDER BY created_at DESC LIMIT 1`, base))
		if err == nil {
			job = running
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("find check items job: %w", err)
		}
		if err := lockUserChecks(ctx, c, userID, maxPending); err != nil {
			return err
		}
		var n int
		if err := c.row(ctx, `SELECT COUNT(*) FROM generation_jobs WHERE kind = 'check_items' AND objective_key = $1`, base).Scan(&n); err != nil {
			return err
		}
		idem := base
		if n > 0 {
			idem = base + "#" + strconv.Itoa(n+1)
		}
		job, err = scanJob(c.row(ctx, `
			INSERT INTO generation_jobs (id, idempotency_key, kind, user_id, concept_id, objective_key, status, attempts, started_at)
			VALUES ($1, $2, 'check_items', $3, NULLIF($4, ''), $5, 'running', 1, NOW())
			RETURNING `+jobColumns, NewID(), idem, userID, conceptID, base))
		if err != nil {
			return fmt.Errorf("insert check items job: %w", err)
		}
		started = true
		return nil
	})
	return job, started, err
}
