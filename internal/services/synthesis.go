package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wooo-jin/punisher/internal/check"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/llm"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/store"
)

// ErrNoSynthesis: not enough checked concepts (3 in 2 areas) or a synthesis this week already (409).
var ErrNoSynthesis = errors.New("종합 문제를 만들 만큼 확인한 개념이 없어요")

const synthesisAttempts = 3

// SynthesisSessionKey is the session key of the user's synthesis of the KST ISO week of now.
func SynthesisSessionKey(userID string, now time.Time) string {
	seed := learning.SynthesisWeekSeed(userID, now)
	return "synthesis:" + seed[strings.LastIndexByte(seed, ':')+1:]
}

type synthesisPlan struct {
	key        string
	conceptID  string
	objectives []store.SynthesisObjective
}

func (p synthesisPlan) objectiveIDs() []string {
	ids := make([]string, 0, len(p.objectives))
	for _, o := range p.objectives {
		ids = append(ids, o.ID)
	}
	return ids
}

// SynthesisDue reports whether the user can start this week's synthesis now
// (eligible and no synthesis session in the last week) and the concept names
// it would mix. An open session of this week counts as due (to continue it).
func (s *Service) SynthesisDue(ctx context.Context, userID string, now time.Time) (bool, error) {
	if open, err := s.Store.LatestCheckSession(ctx, userID, SynthesisSessionKey(userID, now)); err == nil &&
		(open.Status == "preparing" || open.Status == "active") {
		return true, nil
	} else if err != nil && !errors.Is(err, store.ErrNotFound) {
		return false, err
	}
	cands, err := s.Store.SynthesisCandidates(ctx, userID)
	if err != nil {
		return false, err
	}
	last, err := s.Store.LastSynthesisAt(ctx, userID)
	if err != nil {
		return false, err
	}
	return learning.SynthesisEligible(cands, last, now), nil
}

func (s *Service) planSynthesis(ctx context.Context, userID string, now time.Time) (synthesisPlan, error) {
	cands, err := s.Store.SynthesisCandidates(ctx, userID)
	if err != nil {
		return synthesisPlan{}, err
	}
	concepts := map[string]bool{}
	var ids []string
	for _, c := range cands {
		if !concepts[c.ConceptID] {
			concepts[c.ConceptID] = true
			ids = append(ids, c.ConceptID)
		}
	}
	links, err := s.Store.ConceptNeighbors(ctx, ids)
	if err != nil {
		return synthesisPlan{}, err
	}
	pick := learning.PickSynthesis(cands, links, learning.SynthesisWeekSeed(userID, now))
	if len(pick.ConceptIDs) < learning.SynthesisMinConcepts {
		return synthesisPlan{}, ErrNoSynthesis
	}
	objs, err := s.Store.SynthesisObjectives(ctx, userID, pick.ObjectiveIDs)
	if err != nil {
		return synthesisPlan{}, err
	}
	return synthesisPlan{key: SynthesisSessionKey(userID, now), conceptID: pick.ConceptIDs[0], objectives: objs}, nil
}

// startSynthesis opens (or continues) this week's synthesis session; the items
// are written when it starts (v3 §5: made on demand, not queued ahead).
func (s *Service) startSynthesis(ctx context.Context, userID string) (*models.CheckDetail, bool, error) {
	now := s.now()
	key := SynthesisSessionKey(userID, now)
	if open, err := s.Store.LatestCheckSession(ctx, userID, key); err == nil && (open.Status == "preparing" || open.Status == "active") {
		d, err := s.GetCheck(ctx, userID, open.ID)
		return &d, false, err
	}
	due, err := s.SynthesisDue(ctx, userID, now)
	if err != nil {
		return nil, false, err
	}
	if !due {
		return nil, false, ErrNoSynthesis
	}
	plan, err := s.planSynthesis(ctx, userID, now)
	if err != nil {
		return nil, false, err
	}
	apiKey, err := s.checkKey(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	sess, created, err := s.Store.CreateCheckSession(ctx, store.NewCheckSession{UserID: userID, Kind: check.PurposeSynthesis,
		Grader: learning.GraderLLM, ConceptID: plan.conceptID, SessionKey: key, MaxPending: maxPendingChecks})
	if err != nil {
		return nil, false, checkBusy(err)
	}
	if created {
		s.runSynthesisDetached(apiKey, userID, sess.ID, plan)
	}
	d, err := s.GetCheck(ctx, userID, sess.ID)
	return &d, created, err
}

// retrySynthesis regenerates a failed synthesis ('다시 시도') with this week's pick.
func (s *Service) retrySynthesis(ctx context.Context, userID string, sess models.CheckSession) (models.CheckDetail, error) {
	plan, err := s.planSynthesis(ctx, userID, s.now())
	if err != nil {
		return models.CheckDetail{}, err
	}
	apiKey, err := s.checkKey(ctx, userID)
	if err != nil {
		return models.CheckDetail{}, err
	}
	if err := s.Store.RetryCheckSession(ctx, userID, sess.ID, maxPendingChecks); err != nil {
		return models.CheckDetail{}, checkBusy(err)
	}
	s.runSynthesisDetached(apiKey, userID, sess.ID, plan)
	return s.GetCheck(ctx, userID, sess.ID)
}

func (s *Service) runSynthesisDetached(apiKey, userID, sessionID string, plan synthesisPlan) {
	runDetached("synthesis items", sessionID, diagnosticTimeout, func(bg context.Context) error {
		return s.runSynthesisItems(bg, apiKey, userID, sessionID, plan)
	}, s.failPreparing(userID, sessionID, ""))
}

// runSynthesisItems writes the two synthesis items (check.synthesis block,
// check model), stores them and asks the first. Failure fails the session.
func (s *Service) runSynthesisItems(ctx context.Context, apiKey, userID, sessionID string, plan synthesisPlan) error {
	done := context.WithoutCancel(ctx)
	set := s.UserPrompts(ctx, userID)
	sc := check.SynthesisContext{ObjectiveIDs: plan.objectiveIDs(), ConceptOf: map[string]string{}}
	pc := prompts.SynthesisContext{}
	for _, o := range plan.objectives {
		sc.ConceptOf[o.ID] = o.ConceptID
		pc.Objectives = append(pc.Objectives, prompts.DiagnosticObjective{ID: o.ID, ConceptName: o.ConceptName, Text: o.Text})
	}
	items, u, _, err := s.writeItemSet(ctx, apiKey, itemSetCall{model: s.checkModel(), attempts: synthesisAttempts,
		priority: llm.PriorityInteractive, system: set.Text(prompts.BlockCheckSynthesis, nil)},
		func(revisions []string) string { pc.Revisions = revisions; return prompts.Synthesis(pc) },
		func(set check.ItemSet) []string { return check.ValidateSynthesis(set, sc) })
	s.recordUsage(done, userID, "check_items", plan.conceptID, u, err)
	var saved []store.CheckItem
	if err == nil {
		hash := "synthesis"
		if h := set.Hash(prompts.BlockCheckSynthesis); h != "" {
			hash += ":" + h
		}
		saved, err = s.Store.SaveSynthesisItems(done, store.SynthesisItems{ConceptID: plan.conceptID, CreatedFor: userID,
			PromptHash: hash, Model: u.model, PromptVersion: prompts.CheckVersion, Items: check.NormalizeSynthesis(items, plan.objectiveIDs())})
	}
	if err == nil {
		ids := make([]string, 0, len(saved))
		for _, it := range saved {
			ids = append(ids, it.ID)
		}
		ask := store.TurnWrite{ItemID: saved[0].ID, Role: "tutor", Move: check.MoveAsk, Text: saved[0].Prompt}
		_, err = s.Store.ActivateCheckSession(done, userID, sessionID, ids, ask)
	}
	if err != nil {
		if serr := s.Store.SetCheckSessionStatus(done, userID, sessionID, []string{"preparing"}, "failed", err.Error()); serr != nil && !errors.Is(serr, store.ErrCheckClosed) {
			logf("synthesis %s: fail session: %v", sessionID, serr)
		}
	}
	return err
}
