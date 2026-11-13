package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wooo-jin/punisher/internal/curriculum"
	"github.com/wooo-jin/punisher/internal/inquiry"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/llm"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/store"
)

const (
	// MaxInquiryRunes bounds a question.
	MaxInquiryRunes = 1000
	// MaxFollowups / MaxFollowupRunes / MaxQuoteRunes bound a thread (남용 방지).
	MaxFollowups     = 20
	MaxFollowupRunes = 2000
	MaxQuoteRunes    = 1000
	// maxPendingInquiries bounds a user's concurrent generations.
	maxPendingInquiries = 3
	// inquiryDedupeWindow: the same question submitted again while pending returns that row.
	inquiryDedupeWindow = 5 * time.Minute
	// inquiryAttempts: one automatic rewrite when the output does not parse or validate.
	inquiryAttempts = 2
	// inquiryTimeout bounds one explanation (both attempts); StaleInquiryAge is longer so the sweep never ends a live run.
	inquiryTimeout = 6 * time.Minute
	// StaleInquiryAge: a pending row older than this has lost its generation (see store.SweepStaleInquiries).
	StaleInquiryAge = 8 * time.Minute
	// inquiryFailedMessage is all the client sees of a failure (raw errors stay in the DB and logs).
	inquiryFailedMessage = "generation failed"
)

var (
	// ErrInvalidInquiry means the question or follow-up is empty or too long.
	ErrInvalidInquiry = errors.New("question is required (max length exceeded)")
	// ErrInquiryBusy: too many generations at once, or the thread already has one running.
	ErrInquiryBusy = errors.New("다른 해설을 쓰는 중이에요. 끝난 뒤 다시 시도해 주세요")
	// ErrFollowupLimit: the thread reached MaxFollowups.
	ErrFollowupLimit = errors.New("이 질문에서 이어서 물을 수 있는 횟수를 다 썼어요. 새 질문으로 물어봐 주세요")
	// ErrInquiryNotReady: follow-ups need a finished first explanation.
	ErrInquiryNotReady = errors.New("해설이 끝난 뒤 이어서 물을 수 있어요")
)

// StartInquiry stores the question and writes its explanation in the background
// (interactive lane — the user is waiting). The gateway key is resolved now.
func (s *Service) StartInquiry(ctx context.Context, userID, question string) (models.Inquiry, error) {
	question = strings.TrimSpace(question)
	if question == "" || utf8.RuneCountInString(question) > MaxInquiryRunes {
		return models.Inquiry{}, ErrInvalidInquiry
	}
	apiKey, err := s.inquiryKey(ctx, userID)
	if err != nil {
		return models.Inquiry{}, err
	}
	q, created, err := s.Store.CreateInquiry(ctx, userID, question, inquiryDedupeWindow, maxPendingInquiries)
	if err != nil || !created {
		return q, busyErr(err)
	}
	s.inBackground(q.ID, func(bg context.Context) error { return s.runInquiry(bg, apiKey, userID, q) })
	return q, nil
}

// RegenerateInquiry writes a new explanation for the same question ('다시 만들기').
// The previous answer stays visible until the new one is saved.
func (s *Service) RegenerateInquiry(ctx context.Context, userID, id string) (models.Inquiry, error) {
	current, err := s.Store.GetInquiry(ctx, userID, id)
	if err != nil {
		return models.Inquiry{}, err
	}
	if followupPending(current) {
		return models.Inquiry{}, ErrInquiryBusy // 후속 답을 쓰는 동안은 다시 만들지 않는다.
	}
	apiKey, err := s.inquiryKey(ctx, userID)
	if err != nil {
		return models.Inquiry{}, err
	}
	if err := s.checkPendingLimit(ctx, userID); err != nil {
		return models.Inquiry{}, err
	}
	q, started, err := s.Store.SetInquiryPending(ctx, userID, id)
	if err != nil || !started {
		return q, err
	}
	s.inBackground(q.ID, func(bg context.Context) error { return s.runInquiry(bg, apiKey, userID, q) })
	return q, nil
}

// GetInquiry loads a thread with its related/proposed concepts resolved for display.
func (s *Service) GetInquiry(ctx context.Context, userID, id string) (models.Inquiry, error) {
	q, err := s.Store.GetInquiry(ctx, userID, id)
	if err != nil {
		return q, err
	}
	q.Concepts, err = s.Store.InquiryConcepts(ctx, userID, inquiry.MergeIDs(q.ConceptIDs, q.ProposedConceptIDs, len(q.ConceptIDs)+len(q.ProposedConceptIDs)))
	return PublicInquiry(q), err
}

// PublicInquiry hides raw failure text from clients.
func PublicInquiry(q models.Inquiry) models.Inquiry {
	if q.Error != "" {
		q.Error = inquiryFailedMessage
	}
	for i := range q.Followups {
		if q.Followups[i].Error != "" {
			q.Followups[i].Error = inquiryFailedMessage
		}
	}
	return q
}

func (s *Service) inquiryKey(ctx context.Context, userID string) (string, error) {
	if !s.AllowUser(userID) {
		return "", ErrNotOwner
	}
	if err := s.CheckBudget(ctx, userID, false); err != nil {
		return "", err
	}
	return s.Keys.RequestKey(ctx, userID)
}

func (s *Service) checkPendingLimit(ctx context.Context, userID string) error {
	n, err := s.Store.CountPendingInquiries(ctx, userID)
	if err != nil {
		return err
	}
	if n >= maxPendingInquiries {
		return ErrInquiryBusy
	}
	return nil
}

// inBackground runs a generation on the interactive lane, detached from the request.
func (s *Service) inBackground(id string, run func(context.Context) error) {
	runDetached("inquiry", id, inquiryTimeout, run)
}

// runInquiry generates, parses and validates (one rewrite with the problems fed
// back), adds proposed concepts to the map and stores the answer. LLM or
// validation failure is stored as failed (never as an answer).
func (s *Service) runInquiry(ctx context.Context, apiKey, userID string, q models.Inquiry) error {
	done := context.WithoutCancel(ctx) // 생성 시간이 다 돼도 결과·사용량 기록은 남긴다.
	ic, known, err := s.inquiryContext(ctx, userID, q.Question)
	if err != nil {
		return s.Store.FailInquiry(done, userID, q.ID, "", err.Error(), nil)
	}
	system := s.UserPrompts(ctx, userID).Text(prompts.BlockTeacherSystem, s.interactionVars(ctx, userID, ""))
	var u usage
	var tries []store.AttemptRecord
	var answer inquiry.Answer
	var problems []string
	for n := 1; n <= inquiryAttempts; n++ {
		ic.Revisions = problems
		text, gerr := s.complete(ctx, apiKey, llm.MessagesRequest{
			Model: s.Cfg.LectureModel, System: system, User: prompts.Inquiry(ic),
			MaxTokens: 10000, Timeout: 4 * time.Minute, Priority: llm.PriorityInteractive,
		}, &u)
		if gerr != nil {
			err = gerr
			tries = append(tries, store.AttemptRecord{Attempt: n, Result: "llm_error", Message: gerr.Error()})
			break
		}
		var perr error
		if answer, perr = inquiry.Parse(text, known); perr != nil {
			problems = []string{"마크업을 해석할 수 없습니다: " + perr.Error()}
			tries = append(tries, store.AttemptRecord{Attempt: n, Result: "parse_error", Message: perr.Error()})
			continue
		}
		vc := lecture.ValidationContext{KnownConceptIDs: inquiry.KnownWith(known, answer.NewConcepts)}
		if problems = inquiry.Validate(answer, vc); len(problems) > 0 {
			tries = append(tries, store.AttemptRecord{Attempt: n, Result: "invalid", Message: strings.Join(problems, "; ")})
			continue
		}
		tries = append(tries, store.AttemptRecord{Attempt: n, Result: "ok"})
		break
	}
	if err == nil && len(problems) > 0 {
		err = errors.New("explanation failed validation: " + strings.Join(problems, "; "))
	}
	s.recordUsage(done, userID, "inquiry", "", u, err)
	if err != nil {
		return s.Store.FailInquiry(done, userID, q.ID, u.model, err.Error(), tries)
	}
	proposed := s.saveProposed(done, userID, q.ID, answer.NewConcepts)
	inquiry.Relink(&answer.Content, allowedIDs(known, proposed))
	content, err := json.Marshal(answer.Content)
	if err != nil {
		return s.Store.FailInquiry(done, userID, q.ID, u.model, err.Error(), tries)
	}
	return s.Store.AnswerInquiry(done, userID, q.ID, store.InquiryAnswer{
		Content: content, ConceptIDs: answer.ConceptIDs, ProposedConceptIDs: proposed,
		Model: u.model, PromptVersion: prompts.InquiryVersion, Attempts: tries,
	})
}

// saveProposed adds proposed concepts to the user's map and returns the ids the
// user can actually see (a slug taken by someone else's proposal is dropped).
func (s *Service) saveProposed(ctx context.Context, userID, logID string, list []models.ProposedConcept) []string {
	ids := make([]string, 0, len(list))
	for _, p := range list {
		id, err := s.Store.ProposeConcept(ctx, userID, p)
		if err != nil {
			logf("inquiry %s: propose %s: %v", logID, p.ID, err)
			continue
		}
		ids = append(ids, id)
	}
	visible, err := s.Store.InquiryConcepts(ctx, userID, ids)
	if err != nil {
		logf("inquiry %s: proposed visibility: %v", logID, err)
		return []string{}
	}
	out := make([]string, 0, len(visible))
	for _, c := range visible {
		out = append(out, c.ID)
	}
	return out
}

func allowedIDs(known map[string]bool, extra []string) map[string]bool {
	out := make(map[string]bool, len(known)+len(extra))
	for id := range known {
		out[id] = true
	}
	for _, id := range extra {
		out[id] = true
	}
	return out
}

func (s *Service) inquiryContext(ctx context.Context, userID, question string) (prompts.InquiryContext, map[string]bool, error) {
	catalog, err := s.Store.ConceptCatalog(ctx, userID)
	if err != nil {
		return prompts.InquiryContext{}, nil, err
	}
	ic := prompts.InquiryContext{Question: question, Areas: curriculum.Areas}
	known := make(map[string]bool, len(catalog))
	for _, c := range catalog {
		known[c.ID] = true
		ic.Catalog = append(ic.Catalog, prompts.CatalogRef{ID: c.ID, Name: c.Name, Area: c.Area, Learned: learning.IsDone(c.State)})
		if learning.IsDone(c.State) {
			ic.Learned = append(ic.Learned, prompts.ConceptRef{ID: c.ID, Name: c.Name})
		}
	}
	weak, err := s.weakConcepts(ctx, userID)
	if err != nil {
		return prompts.InquiryContext{}, nil, err
	}
	ic.Learned, ic.Revisit = splitLearned(ic.Learned, weak)
	if ic.Misconceptions, err = s.promptMisconceptions(ctx, userID, ""); err != nil {
		return prompts.InquiryContext{}, nil, err
	}
	return ic, known, nil
}

// StartFollowup adds a question to a finished thread and answers it in the background.
func (s *Service) StartFollowup(ctx context.Context, userID, inquiryID, question, quote string) (models.InquiryFollowup, error) {
	question, quote = strings.TrimSpace(question), strings.TrimSpace(quote)
	if question == "" || utf8.RuneCountInString(question) > MaxFollowupRunes || utf8.RuneCountInString(quote) > MaxQuoteRunes {
		return models.InquiryFollowup{}, ErrInvalidInquiry
	}
	q, err := s.Store.GetInquiry(ctx, userID, inquiryID)
	if err != nil {
		return models.InquiryFollowup{}, err
	}
	if q.AnsweredAt == nil {
		return models.InquiryFollowup{}, ErrInquiryNotReady
	}
	if len(q.Followups) >= MaxFollowups {
		return models.InquiryFollowup{}, ErrFollowupLimit
	}
	if q.Status == "pending" || followupPending(q) {
		return models.InquiryFollowup{}, ErrInquiryBusy
	}
	apiKey, err := s.inquiryKey(ctx, userID)
	if err != nil {
		return models.InquiryFollowup{}, err
	}
	if err := s.checkPendingLimit(ctx, userID); err != nil {
		return models.InquiryFollowup{}, err
	}
	f, err := s.Store.CreateFollowup(ctx, userID, inquiryID, question, quote)
	if err != nil {
		return f, busyErr(err)
	}
	s.inBackground(f.ID, func(bg context.Context) error { return s.runFollowup(bg, apiKey, userID, inquiryID, f) })
	return f, nil
}

// RetryFollowup answers a failed follow-up again.
func (s *Service) RetryFollowup(ctx context.Context, userID, inquiryID, id string) (models.InquiryFollowup, error) {
	q, err := s.Store.GetInquiry(ctx, userID, inquiryID)
	if err != nil {
		return models.InquiryFollowup{}, err
	}
	if q.Status == "pending" {
		return models.InquiryFollowup{}, ErrInquiryBusy
	}
	apiKey, err := s.inquiryKey(ctx, userID)
	if err != nil {
		return models.InquiryFollowup{}, err
	}
	if err := s.checkPendingLimit(ctx, userID); err != nil {
		return models.InquiryFollowup{}, err
	}
	f, err := s.Store.SetFollowupPending(ctx, userID, inquiryID, id)
	if err != nil {
		return f, busyErr(err)
	}
	s.inBackground(f.ID, func(bg context.Context) error { return s.runFollowup(bg, apiKey, userID, inquiryID, f) })
	return f, nil
}

func (s *Service) runFollowup(ctx context.Context, apiKey, userID, inquiryID string, f models.InquiryFollowup) error {
	done := context.WithoutCancel(ctx)
	q, err := s.Store.GetInquiry(ctx, userID, inquiryID)
	if err != nil {
		return s.Store.FailFollowup(done, userID, f.ID, "", err.Error())
	}
	ic, known, err := s.inquiryContext(ctx, userID, q.Question)
	if err != nil {
		return s.Store.FailFollowup(done, userID, f.ID, "", err.Error())
	}
	fc, err := followupContext(q, f, ic)
	if err != nil {
		return s.Store.FailFollowup(done, userID, f.ID, "", err.Error())
	}
	system := s.UserPrompts(ctx, userID).Text(prompts.BlockTeacherSystem, s.interactionVars(ctx, userID, ""))
	var u usage
	var answer inquiry.Answer
	var problems []string
	for n := 1; n <= inquiryAttempts; n++ {
		fc.Revisions = problems
		text, gerr := s.complete(ctx, apiKey, llm.MessagesRequest{
			Model: s.Cfg.LectureModel, System: system, User: prompts.InquiryFollowup(fc),
			MaxTokens: 6000, Timeout: 3 * time.Minute, Priority: llm.PriorityInteractive,
		}, &u)
		if gerr != nil {
			err = gerr
			break
		}
		var perr error
		if answer, perr = inquiry.ParseFollowup(text, known); perr != nil {
			problems = []string{"마크업을 해석할 수 없습니다: " + perr.Error()}
			continue
		}
		vc := lecture.ValidationContext{KnownConceptIDs: inquiry.KnownWith(known, answer.NewConcepts)}
		if problems = inquiry.ValidateFollowup(answer, vc); len(problems) == 0 {
			break
		}
	}
	if err == nil && len(problems) > 0 {
		err = errors.New("follow-up failed validation: " + strings.Join(problems, "; "))
	}
	s.recordUsage(done, userID, "inquiry_followup", "", u, err)
	if err != nil {
		return s.Store.FailFollowup(done, userID, f.ID, u.model, err.Error())
	}
	proposed := s.saveProposed(done, userID, f.ID, answer.NewConcepts)
	inquiry.Relink(&answer.Content, allowedIDs(known, proposed))
	section, err := json.Marshal(inquiry.FollowupSection(answer, f.ID))
	if err != nil {
		return s.Store.FailFollowup(done, userID, f.ID, u.model, err.Error())
	}
	return s.Store.AnswerFollowup(done, userID, inquiryID, f.ID, store.FollowupAnswer{
		Section: section, ConceptIDs: answer.ConceptIDs, ProposedConceptIDs: proposed, Model: u.model,
	})
}

func followupPending(q models.Inquiry) bool {
	for _, f := range q.Followups {
		if f.Status == "pending" {
			return true
		}
	}
	return false
}

// busyErr maps the store's busy signal to the user-facing error.
func busyErr(err error) error {
	if errors.Is(err, store.ErrInquiryBusy) {
		return ErrInquiryBusy
	}
	return err
}

// followupContext builds the prompt input from the thread: the first
// explanation and every finished turn before this follow-up.
func followupContext(q models.Inquiry, f models.InquiryFollowup, ic prompts.InquiryContext) (prompts.FollowupContext, error) {
	var content lecture.Content
	if err := json.Unmarshal(q.Content, &content); err != nil {
		return prompts.FollowupContext{}, err
	}
	fc := prompts.FollowupContext{
		Question: f.Question, Quote: f.Quote, Original: q.Question,
		Catalog: ic.Catalog, Learned: ic.Learned, Revisit: ic.Revisit, Misconceptions: ic.Misconceptions, Areas: ic.Areas,
	}
	for _, s := range content.Sections {
		fc.Sections = append(fc.Sections, inquiry.SectionText(s))
	}
	for _, prev := range q.Followups {
		if prev.ID == f.ID {
			break
		}
		if prev.Status != "answered" {
			continue
		}
		var s lecture.Section
		if err := json.Unmarshal(prev.Section, &s); err != nil {
			continue
		}
		fc.Previous = append(fc.Previous, prompts.FollowupQA{Question: prev.Question, Answer: inquiry.SectionText(s)})
	}
	return fc, nil
}
