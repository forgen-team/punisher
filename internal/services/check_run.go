package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wooo-jin/punisher/internal/check"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/store"
	"github.com/wooo-jin/punisher/internal/llm"
	"github.com/wooo-jin/punisher/internal/llmjson"
)

// checkAttempts: one automatic rewrite (problems fed back) when the output does not parse or validate.
const checkAttempts = 2

// heldText is shown when a grade is held: the model answer and '판정 보류'.
func heldText(answer string) string {
	return "판정을 보류했어요(채점이 확실하지 않아 이번 답은 숙달 근거로 쓰지 않아요). 모범답: " + answer
}

// runCheckItems writes the lecture's item set (one rewrite on problems),
// records the tokens on the set's job (kind check_items) and activates the
// session. Failure is recorded as failed, never as items.
func (s *Service) runCheckItems(ctx context.Context, apiKey, userID, sessionID string, cl checkLecture, key store.CheckItemKey, jobID string) error {
	done := context.WithoutCancel(ctx)
	set := s.UserPrompts(ctx, userID)
	ic, sc := checkItemsContext(cl)
	system := set.Text(prompts.BlockCheckItems, s.interactionVars(ctx, userID, cl.info.ID))
	items, u, tries, err := s.writeItemSet(ctx, apiKey, itemSetCall{model: s.Cfg.LectureModel, attempts: checkAttempts,
		priority: llm.PriorityInteractive, system: system},
		func(revisions []string) string { ic.Revisions = revisions; return prompts.CheckItems(ic) },
		func(set check.ItemSet) []string { return check.ValidateItems(set, sc) })
	return s.saveCheckItems(done, userID, sessionID, cl, key, jobID, set, items, u, tries, err)
}

func (s *Service) saveCheckItems(ctx context.Context, userID, sessionID string, cl checkLecture, key store.CheckItemKey, jobID string,
	set prompts.Set, items check.ItemSet, u usage, tries []store.AttemptRecord, err error) error {
	var saved []store.CheckItem
	if err == nil { // 문항을 저장한 뒤에 job 을 끝낸다: 저장이 실패하면 job 도 실패로 남는다.
		saved, err = s.Store.SaveCheckItems(ctx, key, cl.info.ID, jobID, u.model, checkPromptVersion(set),
			check.NormalizeItems(items, cl.objectiveIDs()))
	}
	s.finishItemsJob(ctx, "check "+sessionID, jobID, u, tries, err)
	if err != nil {
		if serr := s.Store.SetCheckSessionStatus(ctx, userID, sessionID, []string{"preparing"}, "failed", checkErrorText(err)); serr != nil && !errors.Is(serr, store.ErrCheckClosed) {
			logf("check %s: fail session: %v", sessionID, serr)
		}
		return err
	}
	_, err = s.activateCheck(ctx, userID, sessionID, saved, cl)
	return err
}

// finishItemsJob records an item generation's job after its items were saved
// (err = the generation's or the save's failure).
func (s *Service) finishItemsJob(ctx context.Context, label, jobID string, u usage, tries []store.AttemptRecord, err error) {
	status := "succeeded"
	if err != nil {
		status = "failed"
	}
	o := u.outcome(status, "", err, "")
	o.Attempts = tries
	if ferr := s.Store.FinishJob(context.WithoutCancel(ctx), jobID, o); ferr != nil {
		logf("%s: finish items job: %v", label, ferr)
	}
}

func checkPromptVersion(set prompts.Set) string {
	if h := set.Hash(prompts.BlockCheckItems); h != "" {
		return prompts.CheckVersion + "+u:" + h
	}
	return prompts.CheckVersion
}

// checkItemsContext renders the lecture body with its anchors for the item prompt.
func checkItemsContext(cl checkLecture) (prompts.CheckItemsContext, check.SetContext) {
	ic := prompts.CheckItemsContext{ConceptName: cl.info.Name, LectureTitle: cl.lec.Title}
	sc := check.SetContext{ObjectiveIDs: cl.objectiveIDs(), Anchors: map[string]bool{}}
	for _, o := range cl.objectives {
		ic.Objectives = append(ic.Objectives, prompts.CheckObjective{ID: o.ID, Text: o.Text})
	}
	for _, sec := range cl.content.Sections {
		var b strings.Builder
		fmt.Fprintf(&b, "### [%s] %s\n", sec.Anchor, sec.Title)
		addAnchor(&ic, sc, sec.Anchor)
		for _, blk := range sec.Blocks {
			addAnchor(&ic, sc, blk.Anchor)
			switch blk.Type {
			case lecture.BlockMarkdown, lecture.BlockMisconception, lecture.BlockCode, lecture.BlockLog:
				fmt.Fprintf(&b, "[%s] %s\n", blk.Anchor, blk.Text)
			}
		}
		ic.Sections = append(ic.Sections, b.String())
	}
	return ic, sc
}

func addAnchor(ic *prompts.CheckItemsContext, sc check.SetContext, anchor string) {
	if anchor != "" && !sc.Anchors[anchor] {
		sc.Anchors[anchor] = true
		ic.Anchors = append(ic.Anchors, anchor)
	}
}

// runCheckTurn grades the current item's latest answer and makes the next move
// (one rewrite when the reply breaks the rules); an item-ending move applies
// the evidence immediately. LLM or validation failure fails the turn (retry).
func (s *Service) runCheckTurn(ctx context.Context, apiKey, userID, sessionID, turnID string) error {
	done := context.WithoutCancel(ctx)
	st, err := s.loadCheckState(ctx, userID, sessionID)
	if err != nil {
		return s.Store.FailCheckTurn(done, userID, turnID, "", err.Error())
	}
	// 채점할 문항은 이 턴의 문항이다. 그사이 문항이 닫혔으면(커서가 넘어감) 채점하지 않는다.
	item := st.current()
	if item == nil || item.ID != turnItemID(st.turns, turnID) {
		return s.Store.FailCheckTurn(done, userID, turnID, "", store.ErrCheckClosed.Error())
	}
	p := progressOf(st.turns, item.ID)
	if len(p.answers) == 0 {
		return s.Store.FailCheckTurn(done, userID, turnID, "", "no answer to grade")
	}
	stage := check.Stage{Answers: len(p.answers), Challenged: p.challenged, Integrative: item.ObjectiveID == "", Last: st.last(),
		Kind: st.itemKind(*item)}
	// 복습 세션은 여러 개념에 걸치므로 개념은 문항에서 정한다(확인 세션은 같은 값).
	reply, u, err := s.tutorReply(ctx, apiKey, userID, item.ConceptID, *item, p, stage)
	s.recordUsage(done, userID, "check_turn", item.ConceptID, u, err)
	if err != nil {
		return s.Store.FailCheckTurn(done, userID, turnID, u.model, checkErrorText(err))
	}
	verdict := check.GradeVerdict(reply.Verdict, reply.Quote, reply.Confidence)
	turn := store.TurnWrite{ID: turnID, UserID: userID, ItemID: item.ID, Role: "tutor", Move: reply.TutorMove, Text: reply.TutorText,
		Verdict: verdict, Aided: len(p.answers) > 1, Grader: learning.GraderLLM, Model: u.model,
		InputTokens: u.input, OutputTokens: u.output,
		Grade: store.GradeRecord{CheckGrade: models.CheckGrade{Met: reply.Met, Missing: reply.Missing, Quote: reply.Quote,
			Misconception: truncateRunes(strings.TrimSpace(reply.Misconception), learning.MaxMisconceptionRunes), Confidence: reply.Confidence}}}
	if verdict == learning.VerdictHeld {
		turn.Text = heldText(item.Rubric.Answer)
	}
	if !check.EndsItem(turn.Move) {
		return s.Store.CompleteCheckTurn(done, turn)
	}
	return s.finishItem(done, userID, st, *item, p.result(verdict, false), p.challenged && verdict == learning.VerdictCorrect, turn, false)
}

// turnItemID is the item of a session turn ("" when the turn is not in the session).
func turnItemID(turns []models.CheckTurn, turnID string) string {
	for _, t := range turns {
		if t.ID == turnID {
			return t.ItemID
		}
	}
	return ""
}

func (s *Service) tutorReply(ctx context.Context, apiKey, userID, conceptID string, item store.CheckItem, p itemProgress, stage check.Stage) (check.TutorReply, usage, error) {
	var u usage
	if len(p.answers) == 0 {
		return check.TutorReply{}, u, errors.New("no answer to grade")
	}
	learner := p.answers[len(p.answers)-1].Text
	tc := prompts.CheckTutorContext{ConceptName: s.interactionVars(ctx, userID, conceptID)["concept_name"],
		Objective: s.objectiveText(ctx, userID, conceptID, item.ObjectiveID), Prompt: item.Prompt, Material: materialText(item),
		Must: item.Rubric.Must, Nice: item.Rubric.Nice, Misconceptions: item.Rubric.Misconceptions, Answer: item.Rubric.Answer,
		LearnerAnswer: learner, Allowed: stage.AllowedMoves()}
	for _, t := range p.dialogue() {
		tc.Dialogue = append(tc.Dialogue, prompts.CheckTurnRef{Role: t.Role, Move: t.Move, Text: t.Text})
	}
	system := s.UserPrompts(ctx, userID).Text(prompts.BlockCheckTutor, s.interactionVars(ctx, userID, conceptID))
	var problems []string
	for n := 1; n <= checkAttempts; n++ {
		tc.Revisions = problems
		text, err := s.complete(ctx, apiKey, llm.MessagesRequest{Model: s.checkModel(), System: system, User: prompts.CheckTutor(tc),
			MaxTokens: 2000, Timeout: 2 * time.Minute, Priority: llm.PriorityInteractive}, &u)
		if err != nil {
			return check.TutorReply{}, u, err
		}
		raw, perr := llmjson.TryParseJSON[check.TutorReply](text)
		if perr != nil {
			problems = []string{"JSON 을 해석할 수 없습니다: " + perr.Error()}
			continue
		}
		reply, probs := check.ValidateTutorReply(raw, stage, item.Rubric.Answer, learner)
		if problems = probs; len(problems) == 0 {
			return reply, u, nil
		}
	}
	return check.TutorReply{}, u, errors.New("tutor reply failed validation: " + strings.Join(problems, "; "))
}

// dialogue is the item's conversation before the latest learner answer.
func (p itemProgress) dialogue() []models.CheckTurn {
	if len(p.answers) == 0 {
		return nil
	}
	cut := p.answers[len(p.answers)-1].Seq
	out := []models.CheckTurn{}
	for _, t := range append(append([]models.CheckTurn{}, p.answers...), p.judgements...) {
		if t.Seq < cut {
			out = append(out, t)
		}
	}
	return sortTurns(out)
}

// finishItem applies the item's evidence (store.FinishCheckItem). A session
// closed meanwhile (abandoned) keeps the turn but applies nothing.
func (s *Service) finishItem(ctx context.Context, userID string, st checkState, item store.CheckItem, r learning.ItemResult,
	integrativeOK bool, turn store.TurnWrite, abandon bool) error {
	turn.ReusedItem = turn.ReusedItem || st.reusedItem(item.ID)
	f := store.ItemFinish{UserID: userID, SessionID: st.sess.ID, Item: item, Result: r, IntegrativeOK: integrativeOK,
		Turn: turn, Abandon: abandon, Intervals: s.reviewIntervals(ctx, userID), Now: s.now(),
		Record: check.Recorded(st.sess.Kind), KnownVerify: st.sess.Kind == check.PurposeKnownVerify}
	newInflow := st.sess.Kind == check.PurposeCheck || st.itemKind(item) == check.PurposeKnownVerify
	if newInflow && item.ObjectiveID != "" && !f.Record {
		f.Hold = s.reviewHold(ctx, userID) // 신규 유입 조절(§2.6): 확인·'이미 앎' 확인에서 처음 복습에 들어오는 목표만
	}
	if !abandon && !st.last() {
		next := st.items[st.sess.Cursor+1]
		f.NextAsk = &store.TurnWrite{ItemID: next.ID, Role: "tutor", Move: check.MoveAsk, Text: next.Prompt}
		if st.sess.Kind == check.PurposeReview {
			seen, err := s.Store.ItemSeenBefore(ctx, userID, next.ID, st.sess.ID)
			if err != nil {
				return err
			}
			f.NextAsk.ReusedItem = seen
		}
	}
	_, err := s.Store.FinishCheckItem(ctx, f)
	if errors.Is(err, store.ErrCheckClosed) && turn.ID != "" {
		// 포기로 닫힌 세션은 답을 남기고, 활성 세션에서 문항이 넘어갔으면 그 채점은 실패로 둔다(근거 없음).
		if sess, serr := s.Store.GetCheckSession(ctx, userID, st.sess.ID); serr == nil && sess.Status == "active" {
			return s.Store.FailCheckTurn(ctx, userID, turn.ID, turn.Model, store.ErrCheckClosed.Error())
		}
		return s.Store.CompleteCheckTurn(ctx, turn)
	}
	return err
}

// runDispute regrades the item's first answer with the other model; an
// accepted dispute restores the item's evidence (store.ResolveDispute).
func (s *Service) runDispute(ctx context.Context, apiKey, userID, sessionID, turnID string) error {
	done := context.WithoutCancel(ctx)
	st, err := s.loadCheckState(ctx, userID, sessionID)
	if err != nil {
		return s.Store.FailCheckTurn(done, userID, turnID, "", err.Error())
	}
	item, judged, first, err := disputeTarget(st, turnID)
	if err != nil {
		return s.Store.FailCheckTurn(done, userID, turnID, "", err.Error())
	}
	reply, u, err := s.regrade(ctx, apiKey, item, first)
	s.recordUsage(done, userID, "check_turn", item.ConceptID, u, err)
	if err != nil {
		return s.Store.FailCheckTurn(done, userID, turnID, u.model, checkErrorText(err))
	}
	accepted := check.DisputeAccepted(reply)
	text := "다시 채점해도 판정은 그대로예요."
	if accepted {
		text = "다시 채점한 결과 맞는 답이었어요. 이 문항의 반영을 되돌렸어요."
	}
	_, err = s.Store.ResolveDispute(done, store.DisputeResolve{UserID: userID, SessionID: sessionID, ObjectiveID: item.ObjectiveID,
		JudgedTurnID: judged.ID, Accepted: accepted, Now: s.now(), Intervals: s.reviewIntervals(done, userID),
		ConflictText: "다시 채점한 결과 맞는 답이었어요. 판정은 바로잡았지만 이후 기록이 있어 상자는 그대로예요.",
		Turn: store.TurnWrite{ID: turnID, ItemID: item.ID, Role: "tutor",
			Move: check.MoveDispute, Text: text, Verdict: check.GradeVerdict(reply.Verdict, reply.Quote, reply.Confidence),
			Grader: learning.GraderRegrade, Model: u.model, InputTokens: u.input, OutputTokens: u.output,
			Grade: store.GradeRecord{CheckGrade: models.CheckGrade{Met: reply.Met, Missing: reply.Missing, Quote: reply.Quote, Confidence: reply.Confidence}}}})
	return err
}

func (s *Service) regrade(ctx context.Context, apiKey string, item store.CheckItem, answer string) (check.RegradeReply, usage, error) {
	var u usage
	user := prompts.CheckRegrade(item.Prompt, item.Rubric.Must, item.Rubric.Nice, item.Rubric.Answer, answer)
	var problems []string
	for n := 1; n <= checkAttempts; n++ {
		prompt := user
		if len(problems) > 0 {
			prompt += "\n## 이전 출력의 문제(고쳐서 다시 출력)\n- " + strings.Join(problems, "\n- ") + "\n"
		}
		text, err := s.complete(ctx, apiKey, llm.MessagesRequest{Model: s.regradeModel(), System: prompts.SystemCheckRegrade,
			User: prompt, MaxTokens: 1500, Timeout: 2 * time.Minute, Priority: llm.PriorityInteractive}, &u)
		if err != nil {
			return check.RegradeReply{}, u, err
		}
		raw, perr := llmjson.TryParseJSON[check.RegradeReply](text)
		if perr != nil {
			problems = []string{"JSON 을 해석할 수 없습니다: " + perr.Error()}
			continue
		}
		reply, probs := check.ValidateRegrade(raw, answer)
		if problems = probs; len(problems) == 0 {
			return reply, u, nil
		}
	}
	return check.RegradeReply{}, u, errors.New("regrade failed validation: " + strings.Join(problems, "; "))
}
