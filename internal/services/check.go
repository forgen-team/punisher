package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wooo-jin/punisher/internal/check"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/store"
)

const (
	// MaxCheckAnswerRunes / MaxDisputeRunes bound learner text.
	MaxCheckAnswerRunes = 2000
	MaxDisputeRunes     = 500
	// maxPendingChecks bounds a user's running tutor turns + item generations (design §4.3).
	maxPendingChecks = 3
	// checkTurnTimeout / checkItemsTimeout bound one run; the stale ages are longer so a sweep never ends a live run.
	checkTurnTimeout  = 3 * time.Minute
	checkItemsTimeout = 6 * time.Minute
	// StaleCheckTurnAge / StaleCheckPrepareAge / AbandonCheckAge drive SweepStaleChecks.
	StaleCheckTurnAge    = 5 * time.Minute
	StaleCheckPrepareAge = 8 * time.Minute
	AbandonCheckAge      = 7 * 24 * time.Hour
	// disputeWindow: a finished session's items can be disputed this long.
	disputeWindow = 24 * time.Hour
)

var (
	// ErrInvalidCheck: bad input (kind, empty/too long text, verdict, a lecture without objectives).
	ErrInvalidCheck = errors.New("invalid check request")
	// ErrCheckBusy: a turn or generation is already running (or too many for the user).
	ErrCheckBusy = errors.New("채점 중이에요. 끝난 뒤 다시 시도해 주세요")
	// ErrCheckState: the action is not possible at this point of the session.
	ErrCheckState = errors.New("지금은 할 수 없는 동작이에요")
	// ErrLectureNotReviewed: the lecture has no pass review yet; it was queued for the backfill review (409).
	ErrLectureNotReviewed = errors.New("검수를 마친 강의만 확인할 수 있어요")
)

// Codes of CheckInputError (the 400 response's error.code).
const (
	CodeAnswerEmpty       = "ANSWER_EMPTY"
	CodeAnswerTooLong     = "ANSWER_TOO_LONG"
	CodeReasonTooLong     = "REASON_TOO_LONG"
	CodeInvalidVerdict    = "INVALID_VERDICT"
	CodeNoObjectives      = "NO_OBJECTIVES"
	CodeInvalidKind       = "INVALID_KIND"
	CodeInvalidConfidence = "INVALID_CONFIDENCE"
)

// CheckInputError is an ErrInvalidCheck with the cause's code.
type CheckInputError struct {
	Code string
	Msg  string
}

func (e *CheckInputError) Error() string { return ErrInvalidCheck.Error() + ": " + e.Msg }
func (e *CheckInputError) Unwrap() error { return ErrInvalidCheck }

func invalidCheck(code, format string, args ...any) error {
	return &CheckInputError{Code: code, Msg: fmt.Sprintf(format, args...)}
}

// checkLecture is the lecture being checked with its active objectives (in lecture order).
type checkLecture struct {
	lec        models.Lecture
	info       models.ConceptInfo
	objectives []models.ObjectiveInfo
	content    lecture.Content
}

func (cl checkLecture) objectiveIDs() []string {
	ids := make([]string, 0, len(cl.objectives))
	for _, o := range cl.objectives {
		ids = append(ids, o.ID)
	}
	return ids
}

func (s *Service) loadCheckLecture(ctx context.Context, userID, lectureID string) (checkLecture, error) {
	lec, err := s.Store.GetLecture(ctx, userID, lectureID)
	if err != nil {
		return checkLecture{}, err
	}
	info, err := s.Store.ConceptInfo(ctx, userID, lec.ConceptID)
	if err != nil {
		return checkLecture{}, err
	}
	cl := checkLecture{lec: lec, info: info}
	if err := json.Unmarshal(lec.Content, &cl.content); err != nil {
		return cl, fmt.Errorf("decode lecture content: %w", err)
	}
	active := map[string]models.ObjectiveInfo{}
	for _, o := range info.Objectives {
		active[o.ID] = o
	}
	for _, id := range lec.ObjectiveIDs {
		if o, ok := active[id]; ok {
			cl.objectives = append(cl.objectives, o)
		}
	}
	if len(cl.objectives) == 0 {
		return cl, invalidCheck(CodeNoObjectives, "lecture has no active objectives")
	}
	return cl, nil
}

func (s *Service) checkModel() string {
	if s.Cfg.CheckModel != "" {
		return s.Cfg.CheckModel
	}
	return s.Cfg.LectureModel
}

func (s *Service) regradeModel() string {
	if s.Cfg.CheckRegradeModel != "" {
		return s.Cfg.CheckRegradeModel
	}
	return s.Cfg.FastModel
}

// checkKey gates LLM check work: owner, the separate daily check budget, the gateway key.
func (s *Service) checkKey(ctx context.Context, userID string) (string, error) {
	if !s.AllowUser(userID) {
		return "", ErrNotOwner
	}
	if s.Cfg.DailyCheckTokenLimit > 0 {
		tokens, err := s.Store.UsageSinceChecks(ctx, userID, learning.KSTDayStart(s.now()))
		if err != nil {
			return "", fmt.Errorf("load check usage: %w", err)
		}
		if tokens >= s.Cfg.DailyCheckTokenLimit {
			return "", fmt.Errorf("%w: %d/%d check tokens", ErrBudgetExceeded, tokens, s.Cfg.DailyCheckTokenLimit)
		}
	}
	return s.Keys.RequestKey(ctx, userID)
}

func (s *Service) checkItemKey(ctx context.Context, userID, lectureID string) store.CheckItemKey {
	k := store.CheckItemKey{LectureID: lectureID, Purpose: check.PurposeCheck}
	if h := s.UserPrompts(ctx, userID).Hash(prompts.BlockCheckItems); h != "" {
		k.PromptHash, k.CreatedFor = h, userID
	}
	return k
}

// StartCheck opens (or returns the open) '확인하기' session of a lecture. A new
// session gets the lecture's cached item set, or generates it in the background.
// A lecture without a pass review is ErrLectureNotReviewed (and moves to the
// front of the backfill review queue).
func (s *Service) StartCheck(ctx context.Context, userID, kind, lectureID string) (models.CheckDetail, bool, error) {
	if kind != check.PurposeCheck {
		return models.CheckDetail{}, false, invalidCheck(CodeInvalidKind, "kind must be check")
	}
	cl, err := s.loadCheckLecture(ctx, userID, lectureID)
	if err != nil {
		return models.CheckDetail{}, false, err
	}
	if !cl.lec.Reviewed { // 문항은 검수 통과 기록이 있는 강의에서만 만든다(v2.2 §2.1) — 소급 검수 대기열 앞으로.
		if err := s.Store.RequestLectureReview(ctx, cl.lec.ID); err != nil {
			return models.CheckDetail{}, false, err
		}
		return models.CheckDetail{}, false, ErrLectureNotReviewed
	}
	apiKey, err := s.checkKey(ctx, userID)
	if err != nil {
		return models.CheckDetail{}, false, err
	}
	sess, created, err := s.Store.CreateCheckSession(ctx, store.NewCheckSession{UserID: userID, Kind: kind,
		Grader: learning.GraderLLM, ConceptID: cl.lec.ConceptID, LectureID: cl.lec.ID, SessionKey: "lecture:" + cl.lec.ID})
	if err != nil {
		return models.CheckDetail{}, false, err
	}
	if sess.Status == "preparing" {
		if err := s.prepareCheck(ctx, apiKey, userID, sess.ID, cl); err != nil {
			return models.CheckDetail{}, created, checkBusy(err)
		}
	}
	d, err := s.GetCheck(ctx, userID, sess.ID)
	return d, created, err
}

// prepareCheck activates the session with the cached set, or starts the set's
// single generation (a generation already running for the key is awaited).
func (s *Service) prepareCheck(ctx context.Context, apiKey, userID, sessionID string, cl checkLecture) error {
	key := s.checkItemKey(ctx, userID, cl.lec.ID)
	items, err := s.Store.ActiveCheckItems(ctx, userID, key)
	if err != nil {
		return err
	}
	if ok, err := s.activateCheck(ctx, userID, sessionID, items, cl); ok || err != nil {
		return err
	}
	job, started, err := s.Store.StartCheckItemsJob(ctx, userID, cl.lec.ConceptID, key, maxPendingChecks)
	if err != nil || !started {
		return err
	}
	runDetached("check items", sessionID, checkItemsTimeout, func(bg context.Context) error {
		return s.runCheckItems(bg, apiKey, userID, sessionID, cl, key, job.ID)
	}, s.failPreparing(userID, sessionID, job.ID))
	return nil
}

// activateCheck picks the session's items from a set (each objective's check
// item in lecture order, then the integrative item) and asks the first one;
// ok=false when the set is incomplete.
func (s *Service) activateCheck(ctx context.Context, userID, sessionID string, items []store.CheckItem, cl checkLecture) (bool, error) {
	byObjective := map[string]store.CheckItem{}
	var integrative *store.CheckItem
	for i, it := range items {
		switch {
		case it.Status != "active" || it.Purpose != check.PurposeCheck:
		case it.ObjectiveID == "":
			integrative = &items[i]
		default:
			byObjective[it.ObjectiveID] = it
		}
	}
	ordered := []store.CheckItem{}
	for _, id := range cl.objectiveIDs() {
		it, ok := byObjective[id]
		if !ok {
			return false, nil
		}
		ordered = append(ordered, it)
	}
	if integrative == nil {
		return false, nil
	}
	ordered = append(ordered, *integrative)
	ids := make([]string, 0, len(ordered))
	for _, it := range ordered {
		ids = append(ids, it.ID)
	}
	ask := store.TurnWrite{ItemID: ordered[0].ID, Role: "tutor", Move: check.MoveAsk, Text: ordered[0].Prompt}
	_, err := s.Store.ActivateCheckSession(ctx, userID, sessionID, ids, ask)
	return true, err
}

// GetCheck returns a session with its items, turns and the objectives' mastery.
// A preparing session whose set became ready (another session generated it) is activated here.
func (s *Service) GetCheck(ctx context.Context, userID, id string) (models.CheckDetail, error) {
	st, err := s.loadCheckState(ctx, userID, id)
	if err != nil {
		return models.CheckDetail{}, err
	}
	if st.sess.Status == "preparing" && st.sess.Kind == check.PurposeCheck && st.sess.LectureID != "" {
		if st, err = s.attachReadyItems(ctx, userID, st); err != nil {
			return models.CheckDetail{}, err
		}
	}
	return s.checkDetail(ctx, userID, st)
}

func (s *Service) attachReadyItems(ctx context.Context, userID string, st checkState) (checkState, error) {
	cl, err := s.loadCheckLecture(ctx, userID, st.sess.LectureID)
	if err != nil {
		return st, err
	}
	items, err := s.Store.ActiveCheckItems(ctx, userID, s.checkItemKey(ctx, userID, cl.lec.ID))
	if err != nil {
		return st, err
	}
	if ok, err := s.activateCheck(ctx, userID, st.sess.ID, items, cl); !ok || err != nil {
		return st, err
	}
	return s.loadCheckState(ctx, userID, st.sess.ID)
}

func checkBusy(err error) error {
	if errors.Is(err, store.ErrCheckBusy) {
		return ErrCheckBusy
	}
	if errors.Is(err, store.ErrCheckClosed) {
		return ErrCheckState
	}
	return err
}
