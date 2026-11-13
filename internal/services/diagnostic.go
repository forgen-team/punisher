package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wooo-jin/punisher/internal/check"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/store"
	"github.com/wooo-jin/punisher/internal/llm"
)

// diagnosticTimeout bounds one diagnostic generation (FastModel, concept and objectives only).
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
	}
	return nil, false, invalidCheck(CodeInvalidKind, "kind must be check|pretest|known_verify|placement")
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
	info, err := s.Store.ConceptInfo(ctx, userID, conceptID)
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
		ConceptID: plan.conceptID, LectureID: plan.lectureID, SessionKey: plan.key})
	if err != nil {
		return nil, false, err
	}
	if created {
		s.runDiagnosticDetached(apiKey, userID, sess.ID, plan)
	}
	d, err := s.GetCheck(ctx, userID, sess.ID)
	return &d, created, err
}

func (s *Service) runDiagnosticDetached(apiKey, userID, sessionID string, plan diagnosticPlan) {
	runDetached("diagnostic items", sessionID, diagnosticTimeout, func(bg context.Context) error {
		return s.runDiagnosticItems(bg, apiKey, userID, sessionID, plan)
	})
}

// runDiagnosticItems writes the plan's items with the check.pretest block
// (FastModel), records the tokens (kind check_items), stores the items and
// asks the first one. Failure fails the session, never stores items.
func (s *Service) runDiagnosticItems(ctx context.Context, apiKey, userID, sessionID string, plan diagnosticPlan) error {
	done := context.WithoutCancel(ctx)
	set := s.UserPrompts(ctx, userID)
	conceptID := plan.conceptID
	if conceptID == "" && len(plan.objectives) > 0 {
		conceptID = conceptOf(plan.objectives[0].ID)
	}
	dc := prompts.DiagnosticContext{Purpose: plan.kind, Objectives: plan.objectives, Count: check.PretestCount(len(plan.objectives))}
	vc := check.DiagnosticContext{Purpose: plan.kind, ObjectiveIDs: plan.objectiveIDs()}
	items, u, _, err := s.writeItemSet(ctx, apiKey, itemSetCall{model: s.fastModel(), attempts: plan.attempts,
		priority: llm.PriorityInteractive, system: set.Text(prompts.BlockCheckPretest, s.interactionVars(ctx, userID, conceptID))},
		func(revisions []string) string { dc.Revisions = revisions; return prompts.Diagnostic(dc) },
		func(set check.ItemSet) []string { return check.ValidateDiagnostic(set, vc) })
	s.recordUsage(done, userID, "check_items", conceptID, u, err)
	var saved []store.CheckItem
	if err == nil {
		saved, err = s.Store.SaveDiagnosticItems(done, store.DiagnosticItems{PromptHash: set.Hash(prompts.BlockCheckPretest),
			CreatedFor: userID, Model: u.model, PromptVersion: diagnosticPromptVersion(set), Items: check.NormalizeDiagnostic(items)})
	}
	if err == nil {
		saved = orderByObjectives(saved, plan.objectiveIDs())
		ids := make([]string, 0, len(saved))
		for _, it := range saved {
			ids = append(ids, it.ID)
		}
		ask := store.TurnWrite{ItemID: saved[0].ID, Role: "tutor", Move: check.MoveAsk, Text: saved[0].Prompt}
		_, err = s.Store.ActivateCheckSession(done, userID, sessionID, ids, ask)
	}
	if err != nil {
		if serr := s.Store.SetCheckSessionStatus(done, userID, sessionID, []string{"preparing"}, "failed", checkErrorText(err)); serr != nil && !errors.Is(serr, store.ErrCheckClosed) {
			logf("diagnostic %s: fail session: %v", sessionID, serr)
		}
	}
	return err
}

// conceptOf is the concept of an objective id (objective ids are "<concept>#<n>").
func conceptOf(objectiveID string) string {
	concept, _, _ := strings.Cut(objectiveID, "#")
	return concept
}

// orderByObjectives keeps the plan's asking order (placement easy → hard);
// items of the same objective keep their generated order.
func orderByObjectives(items []store.CheckItem, objectiveIDs []string) []store.CheckItem {
	out := make([]store.CheckItem, 0, len(items))
	for _, id := range objectiveIDs {
		for _, it := range items {
			if it.ObjectiveID == id {
				out = append(out, it)
			}
		}
	}
	return out
}

func diagnosticPromptVersion(set prompts.Set) string {
	if h := set.Hash(prompts.BlockCheckPretest); h != "" {
		return prompts.CheckVersion + "+u:" + h
	}
	return prompts.CheckVersion
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
	if err := s.Store.SetCheckSessionStatus(ctx, userID, sess.ID, []string{"failed"}, "preparing", ""); err != nil {
		return models.CheckDetail{}, checkBusy(err)
	}
	s.runDiagnosticDetached(apiKey, userID, sess.ID, plan)
	return s.GetCheck(ctx, userID, sess.ID)
}
