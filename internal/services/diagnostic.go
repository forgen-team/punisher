package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
)

// diagnosticTimeout bounds one diagnostic generation (concept and objectives only; model by diagnosticModel).
const diagnosticTimeout = 3 * time.Minute

// pretestAttempts: the pretest is one FastModel call (design §5 P3); the
// verification and the placement, which the learner waits for, get one rewrite.
const pretestAttempts = 1

// Codes of the P3 400 / 409 responses.
const (
	CodeMissingTarget  = "MISSING_TARGET"
	CodeChoiceRequired = "CHOICE_REQUIRED"
	CodeInvalidChoice  = "INVALID_CHOICE"
)

// ErrNoPlacement: the chapter has fewer than learning.MinPlacementItems concepts left to diagnose.
var ErrNoPlacement = errors.New("배치 진단할 개념이 부족해요")

// DiagnosticRequest is POST /v1/checks for the P3 kinds (design §5 P3).
type DiagnosticRequest struct {
	Kind string
	// LectureID (기존 강의를 처음 열 때) or JobID (즉석 강의 생성 중): pretest.
	LectureID string
	JobID     string
	// ConceptID: known_verify. ChapterID: placement.
	ConceptID string
	ChapterID string
}

// diagnosticPlan is what one diagnostic session asks about.
type diagnosticPlan struct {
	kind      string
	grader    string
	conceptID string
	lectureID string
	key       string
	// objectives in asking order (placement: easy → hard).
	objectives []prompts.DiagnosticObjective
	attempts   int
	// openKeys: open sessions of these keys are continued (pretest).
	openKeys []string
	// opened: the lecture was read before (a pretest is only for the first open).
	opened bool
}

func (p diagnosticPlan) objectiveIDs() []string {
	ids := make([]string, 0, len(p.objectives))
	for _, o := range p.objectives {
		ids = append(ids, o.ID)
	}
	return ids
}

func (s *Service) fastModel() string {
	if s.Cfg.FastModel != "" {
		return s.Cfg.FastModel
	}
	return s.Cfg.LectureModel
}

// diagnosticModel: only the pretest (예측 질문, no evidence) uses FastModel; the
// '이미 앎' verification and the placement change mastery, so they use
// LectureModel (quality design v2.2 §2.3).
func (s *Service) diagnosticModel(kind string) string {
	if kind == check.PurposePretest {
		return s.fastModel()
	}
	return s.Cfg.LectureModel
}

// StartDiagnostic opens a pretest, '이미 앎' verification or placement session
// (an open one of the same key is returned). nil detail = no pretest now
// (turned off, already predicted, lecture read before, or no LLM access).
func (s *Service) StartDiagnostic(ctx context.Context, userID string, req DiagnosticRequest) (*models.CheckDetail, bool, error) {
	switch req.Kind {
	case check.PurposePretest:
		return s.startPretest(ctx, userID, req)
	case check.PurposeKnownVerify:
		plan, err := s.knownVerifyPlan(ctx, userID, req.ConceptID)
		if err != nil {
			return nil, false, err
		}
		return s.createDiagnostic(ctx, userID, plan)
	case check.PurposePlacement:
		plan, err := s.placementPlan(ctx, userID, req.ChapterID)
		if err != nil {
			return nil, false, err
		}
		return s.createDiagnostic(ctx, userID, plan)
	case check.PurposeSynthesis:
		return s.startSynthesis(ctx, userID)
	}
	return nil, false, invalidCheck(CodeInvalidKind, "kind must be check|pretest|known_verify|placement|synthesis")
}

// startPretest: the prediction questions of a lecture start — at most once per
// objective, only before the lecture is first read, off with pretest_enabled.
// The open pretest of the concept (started on the instant-stream screen) is
// continued on the lecture it produced.
func (s *Service) startPretest(ctx context.Context, userID string, req DiagnosticRequest) (*models.CheckDetail, bool, error) {
	if !s.learningSettings(ctx, userID).PretestEnabled || !s.AllowUser(userID) {
		return nil, false, nil
	}
	plan, err := s.pretestPlan(ctx, userID, req)
	if err != nil {
		return nil, false, err
	}
	for _, key := range plan.openKeys {
		open, err := s.Store.LatestCheckSession(ctx, userID, key)
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, false, err
		}
		if open.Status == "preparing" || open.Status == "active" {
			d, err := s.GetCheck(ctx, userID, open.ID)
			return &d, false, err
		}
	}
	if plan.opened {
		return nil, false, nil
	}
	seen, err := s.Store.PretestSeen(ctx, userID, plan.objectiveIDs())
	if err != nil || seen {
		return nil, false, err
	}
	return s.createDiagnostic(ctx, userID, plan)
}

func (s *Service) pretestPlan(ctx context.Context, userID string, req DiagnosticRequest) (diagnosticPlan, error) {
	p := diagnosticPlan{kind: check.PurposePretest, grader: learning.GraderNone, attempts: pretestAttempts}
	var ids []string
	switch {
	case req.LectureID != "":
		lec, err := s.Store.GetLecture(ctx, userID, req.LectureID)
		if err != nil {
			return p, err
		}
		p.conceptID, p.lectureID, ids, p.opened = lec.ConceptID, lec.ID, lec.ObjectiveIDs, lec.Progress != nil
		p.key = "pretest:" + lec.ID
		p.openKeys = []string{p.key, "pretest:" + lec.ConceptID}
	case req.JobID != "":
		job, err := s.Store.GetJob(ctx, userID, req.JobID)
		if err != nil {
			return p, err
		}
		p.conceptID, ids = job.ConceptID, strings.Split(job.ObjectiveKey, ",")
		p.key = "pretest:" + job.ConceptID
		p.openKeys = []string{p.key}
	default:
		return p, invalidCheck(CodeMissingTarget, "lectureId or jobId is required")
	}
	objectives, err := s.diagnosticObjectives(ctx, userID, p.conceptID, ids)
	if err != nil {
		return p, err
	}
	p.objectives = objectives[:min(len(objectives), check.MaxPretestItems)]
	return p, nil
}

// knownVerifyPlan: one item per active objective of the concept (design §5 P3).
func (s *Service) knownVerifyPlan(ctx context.Context, userID, conceptID string) (diagnosticPlan, error) {
	if conceptID == "" {
		return diagnosticPlan{}, invalidCheck(CodeMissingTarget, "conceptId is required")
	}
	objectives, err := s.diagnosticObjectives(ctx, userID, conceptID, nil)
	return diagnosticPlan{kind: check.PurposeKnownVerify, grader: learning.GraderLLM, conceptID: conceptID,
		key: "known_verify:" + conceptID, objectives: objectives, attempts: checkAttempts}, err
}

// placementPlan: 3~5 of the chapter's not-yet-learned concepts, easy → hard,
// one item each on its first objective (learning.PickPlacement).
func (s *Service) placementPlan(ctx context.Context, userID, chapterID string) (diagnosticPlan, error) {
	if chapterID == "" {
		return diagnosticPlan{}, invalidCheck(CodeMissingTarget, "chapterId is required")
	}
	if !validID(chapterID) {
		return diagnosticPlan{}, store.ErrNotFound
	}
	list, err := s.Store.PlacementConcepts(ctx, userID, chapterID)
	if err != nil {
		return diagnosticPlan{}, err
	}
	picks := learning.PickPlacement(list)
	if len(picks) == 0 {
		return diagnosticPlan{}, ErrNoPlacement
	}
	p := diagnosticPlan{kind: check.PurposePlacement, grader: learning.GraderNone, key: "placement:" + chapterID, attempts: checkAttempts}
	for _, c := range picks {
		objectives, err := s.diagnosticObjectives(ctx, userID, c.ConceptID, []string{c.ObjectiveID})
		if err != nil {
			return p, err
		}
		p.objectives = append(p.objectives, objectives...)
	}
	return p, nil
}

// diagnosticObjectives are the concept's active objectives (only ids when
// given), in curriculum order, with the concept name.
func (s *Service) diagnosticObjectives(ctx context.Context, userID, conceptID string, ids []string) ([]prompts.DiagnosticObjective, error) {
	info, err := s.Store.ConceptInfo(ctx, userID, conceptID, s.Store.LocaleFor(ctx, userID))
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[strings.TrimSpace(id)] = true
	}
	out := []prompts.DiagnosticObjective{}
	for _, o := range info.Objectives {
		if len(ids) == 0 || want[o.ID] {
			out = append(out, prompts.DiagnosticObjective{ID: o.ID, ConceptName: info.Name, Text: o.Text})
		}
	}
	if len(out) == 0 {
		return nil, invalidCheck(CodeNoObjectives, "concept %s has no active objectives to diagnose", conceptID)
	}
	return out, nil
}

// createDiagnostic opens the plan's session and writes its items in the
// background. A pretest without LLM access (budget, key) is skipped, not an error.
func (s *Service) createDiagnostic(ctx context.Context, userID string, plan diagnosticPlan) (*models.CheckDetail, bool, error) {
	apiKey, err := s.checkKey(ctx, userID)
	if err != nil {
		if plan.kind == check.PurposePretest {
			logf("pretest for %s skipped: %v", userID, err)
			return nil, false, nil
		}
		return nil, false, err
	}
	sess, created, err := s.Store.CreateCheckSession(ctx, store.NewCheckSession{UserID: userID, Kind: plan.kind, Grader: plan.grader,
		ConceptID: plan.conceptID, LectureID: plan.lectureID, SessionKey: plan.key, MaxPending: maxPendingChecks})
	if errors.Is(err, store.ErrCheckBusy) && plan.kind == check.PurposePretest {
		logf("pretest for %s skipped: %v", userID, err) // 예측 질문은 없어도 강의는 그대로 열린다.
		return nil, false, nil
	}
	if err != nil {
		return nil, false, checkBusy(err)
	}
	if created {
		s.runDiagnosticDetached(apiKey, userID, sess.ID, plan)
	}
	d, err := s.GetCheck(ctx, userID, sess.ID)
	return &d, created, err
}

// retryDiagnostic regenerates a failed verification or placement ('다시 시도');
// a failed pretest is not retried (the lecture goes on without it).
func (s *Service) retryDiagnostic(ctx context.Context, userID string, sess models.CheckSession) (models.CheckDetail, error) {
	var plan diagnosticPlan
	var err error
	switch sess.Kind {
	case check.PurposeKnownVerify:
		plan, err = s.knownVerifyPlan(ctx, userID, sess.ConceptID)
	case check.PurposePlacement:
		plan, err = s.placementPlan(ctx, userID, chapterOfKey(sess.SessionKey))
	case check.PurposeSynthesis:
		return s.retrySynthesis(ctx, userID, sess)
	default:
		return models.CheckDetail{}, ErrCheckState
	}
	if err != nil {
		return models.CheckDetail{}, err
	}
	apiKey, err := s.checkKey(ctx, userID)
	if err != nil {
		return models.CheckDetail{}, err
	}
	if err := s.Store.RetryCheckSession(ctx, userID, sess.ID, maxPendingChecks); err != nil {
		// 같은 키로 이미 열린 세션(다른 탭에서 새로 시작)이 있으면 그 세션을 이어 간다.
		if open, oerr := s.Store.LatestCheckSession(ctx, userID, sess.SessionKey); oerr == nil && open.ID != sess.ID &&
			(open.Status == "preparing" || open.Status == "active") {
			return s.GetCheck(ctx, userID, open.ID)
		}
		return models.CheckDetail{}, checkBusy(err)
	}
	s.runDiagnosticDetached(apiKey, userID, sess.ID, plan)
	return s.GetCheck(ctx, userID, sess.ID)
}
