package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
)

// Reading threads (learner-level design ④): lecture_questions rows with a
// quote (root), parent_id (follow-ups, always the root) and mode.

// ThreadRequest starts a reading thread on a passage. Validated by the caller
// (anchor ≤ 200, quote NormalizeSelection, mode, question required for mode=question).
type ThreadRequest struct {
	Anchor   string
	Quote    string
	Mode     string
	Question string
}

// ErrNotThread: follow-ups and deep lectures need a lecture question (not an
// alt explanation request).
var ErrNotThread = errors.New("only lecture questions have threads")

// DeepLectureQueryRunes bounds the instant-lecture query built from a thread
// (the instant endpoint's query limit).
const DeepLectureQueryRunes = 500

// threadDedupeWindow: the same thread turn submitted again within this window
// (double tap) returns the entry already being answered instead of a second
// entry and a second LLM call.
const threadDedupeWindow = 20 * time.Second

// StartThread creates the root of a reading thread and answers it synchronously
// on the short lane: easier/deeper explain the passage (explain block), a
// question is answered on the QA block with the passage included. Budget and
// key are checked before the entry exists (a refusal leaves no failed entry);
// LLM failure is stored as status=failed, never as an answer. A double submit
// returns the first entry as it is (pending or answered).
func (s *Service) StartThread(ctx context.Context, userID string, lec models.Lecture, req ThreadRequest) (models.Question, error) {
	apiKey, err := s.threadKey(ctx, userID)
	if err != nil {
		return models.Question{}, err
	}
	q, created, err := s.Store.CreateThreadQuestion(ctx, userID, models.Question{
		LectureID: lec.ID, ConceptID: lec.ConceptID, Anchor: req.Anchor, Kind: "question",
		Question: threadQuestionText(req.Mode, req.Question, lec.Language), Quote: req.Quote, Mode: req.Mode,
	}, threadDedupeWindow)
	if err != nil || !created {
		return q, err
	}
	if req.Mode == learning.ThreadQuestion {
		qc, err := s.qaContext(ctx, userID, lec, req.Anchor, false)
		if err != nil {
			return s.failQuestion(ctx, q.ID, "", err)
		}
		return s.answerWithKey(ctx, userID, apiKey, q, qc, prompts.ThreadAsk(qc.prompt, prompts.Thread{
			Quote: req.Quote, Mode: req.Mode, Question: req.Question,
		}))
	}
	return s.answerQuote(ctx, userID, apiKey, q, lec, req.Anchor, req.Quote, req.Mode)
}

// threadKey checks the budget and resolves the gateway key of a thread turn
// (before its entry is created, like RequestAltExplanation).
func (s *Service) threadKey(ctx context.Context, userID string) (string, error) {
	if err := s.CheckBudget(ctx, userID, false); err != nil {
		return "", err
	}
	return s.Keys.RequestKey(ctx, userID)
}

// answerQuote answers a root easier/deeper entry with the passage explanation prompt.
func (s *Service) answerQuote(ctx context.Context, userID, apiKey string, q models.Question, lec models.Lecture, anchor, quote, mode string) (models.Question, error) {
	pc, lang, err := s.explainContext(ctx, userID, lec, anchor)
	if err != nil {
		return s.failQuestion(ctx, q.ID, "", err)
	}
	text, model, err := s.explainWithKey(ctx, userID, apiKey, lec.ConceptID, pc, lang, prompts.ExplainQuote(pc, quote, mode))
	if err != nil {
		return s.failQuestion(ctx, q.ID, model, err)
	}
	return s.saveAnswer(ctx, q.ID, text, model, nil, nil)
}

// Followup adds a turn to the thread of questionID (a root or any follow-up;
// the new entry's parent is always the root) and answers it with the passage
// and the thread's earlier turns. mode easier/deeper re-explains the previous
// answer. Budget/key first and double-submit dedupe as in StartThread.
func (s *Service) Followup(ctx context.Context, userID, questionID, mode, question string) (models.Question, error) {
	thread, lec, err := s.loadThread(ctx, userID, questionID)
	if err != nil {
		return models.Question{}, err
	}
	apiKey, err := s.threadKey(ctx, userID)
	if err != nil {
		return models.Question{}, err
	}
	root := thread[0]
	q, created, err := s.Store.CreateThreadQuestion(ctx, userID, models.Question{
		LectureID: lec.ID, ConceptID: lec.ConceptID, Anchor: root.Anchor, Kind: "question",
		Question: threadQuestionText(mode, question, lec.Language), ParentID: root.ID, Mode: mode,
	}, threadDedupeWindow)
	if err != nil || !created {
		return q, err
	}
	qc, err := s.qaContext(ctx, userID, lec, root.Anchor, false)
	if err != nil {
		return s.failQuestion(ctx, q.ID, "", err)
	}
	return s.answerWithKey(ctx, userID, apiKey, q, qc, prompts.ThreadAsk(qc.prompt, prompts.Thread{
		Quote: root.Quote, Turns: threadTurns(thread), Mode: mode, Question: question,
	}))
}

// DeepLecture starts an instant lecture from a thread ('이 대화로 심화 강의
// 만들기'): the query is the passage plus the reader's questions (no LLM call
// to build it), linked under the thread's lecture like '이 개념 더 깊게'.
func (s *Service) DeepLecture(ctx context.Context, userID, questionID string) (InstantResponse, error) {
	thread, lec, err := s.loadThread(ctx, userID, questionID)
	if err != nil {
		return InstantResponse{}, err
	}
	query := DeepLectureQuery(thread)
	if query == "" {
		query = truncateRunes(lec.Title, DeepLectureQueryRunes)
	}
	resp, err := s.StartInstant(ctx, userID, InstantRequest{
		Query: query, AutoStart: true, ParentLectureID: lec.ID, ParentAnchor: thread[0].Anchor,
	})
	if err != nil || !pointsBack(resp, lec) {
		return resp, err
	}
	// The thread is about the lecture's own concept, whose lecture already exists:
	// returning it would just reopen what the learner is reading. Explain the
	// confusion itself instead (question explanation, written in the background).
	inq, err := s.StartInquiry(ctx, userID, deepInquiryQuestion(lec, query))
	if err != nil {
		// Unwrapped: the handler maps ErrInquiryBusy (409), budget (429) etc. like POST /v1/inquiries.
		return InstantResponse{}, err
	}
	return InstantResponse{Status: InstantInquiry, ConceptID: lec.ConceptID, InquiryID: inq.ID}, nil
}

// InstantInquiry: DeepLecture answered with a question explanation (see InquiryID).
const InstantInquiry = "inquiry"

// pointsBack reports whether an instant result just leads back to lec's concept.
func pointsBack(resp InstantResponse, lec models.Lecture) bool {
	return resp.Status == InstantExisting && (resp.LectureID == lec.ID || resp.ConceptID == lec.ConceptID)
}

// deepInquiryQuestion frames the thread as a question explanation request, in
// the lecture's language (like the other stored request texts).
func deepInquiryQuestion(lec models.Lecture, query string) string {
	format := "강의 「%s」를 읽다가 막힌 부분을 깊게 풀어 주세요. %s"
	if learning.NormalizeLocale(lec.Language) == learning.LocaleEN {
		format = "I got stuck reading the lecture “%s” — please explain this in depth. %s"
	}
	return truncateRunes(fmt.Sprintf(format, lec.Title, query), MaxInquiryRunes)
}

// loadThread loads the whole thread of questionID (root first) and its lecture.
// ErrNotFound for another user's question or a deleted lecture.
func (s *Service) loadThread(ctx context.Context, userID, questionID string) ([]models.Question, models.Lecture, error) {
	q, err := s.Store.GetQuestion(ctx, userID, questionID)
	if err != nil {
		return nil, models.Lecture{}, err
	}
	rootID := q.ID
	if q.ParentID != "" {
		rootID = q.ParentID
	}
	thread, err := s.Store.ListThread(ctx, userID, rootID)
	if err != nil {
		return nil, models.Lecture{}, err
	}
	root := thread[0]
	if root.Kind != "question" {
		return nil, models.Lecture{}, ErrNotThread
	}
	if root.LectureID == "" {
		return nil, models.Lecture{}, store.ErrNotFound
	}
	lec, err := s.Store.GetLecture(ctx, userID, root.LectureID)
	if err != nil {
		return nil, models.Lecture{}, err
	}
	return thread, lec, nil
}

// threadTurns are the answered turns of a thread, oldest first (prompt history).
func threadTurns(thread []models.Question) []prompts.QATurn {
	turns := []prompts.QATurn{}
	for _, t := range thread {
		if t.Status == "answered" {
			turns = append(turns, prompts.QATurn{Question: t.Question, Answer: t.Answer})
		}
	}
	return turns
}

// DeepLectureQuery composes the instant query of a thread: the quoted passage
// (whitespace collapsed, ≤ 200 runes) and the reader's own questions
// (mode=question, ≤ 120 runes each), cut to DeepLectureQueryRunes.
func DeepLectureQuery(thread []models.Question) string {
	var parts []string
	if len(thread) > 0 {
		if quote := collapseSpace(thread[0].Quote); quote != "" {
			parts = append(parts, "“"+truncateRunes(quote, 200)+"”")
		}
	}
	for _, t := range thread {
		if t.Mode == learning.ThreadQuestion || t.Mode == "" {
			if q := collapseSpace(t.Question); q != "" {
				parts = append(parts, truncateRunes(q, 120))
			}
		}
	}
	return truncateRunes(strings.Join(parts, " — "), DeepLectureQueryRunes)
}

func collapseSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

// threadQuestionText is the stored question of a thread entry: the reader's
// words, or for easier/deeper without words the request in the lecture's
// language (like the '다른 방식으로 설명' entry).
func threadQuestionText(mode, question, lang string) string {
	if q := strings.TrimSpace(question); q != "" || mode == learning.ThreadQuestion {
		return q
	}
	en := learning.NormalizeLocale(lang) == learning.LocaleEN
	switch {
	case mode == learning.ThreadEasier && en:
		return "Please explain this more simply."
	case mode == learning.ThreadEasier:
		return "더 쉽게 설명해 주세요."
	case en:
		return "Please explain this in more depth."
	}
	return "더 깊게 설명해 주세요."
}

// LevelSuggestion is the lower-level suggestion of GET /v1/me/settings.
type LevelSuggestion struct {
	To     string `json:"to"`
	Reason string `json:"reason"`
}

// SuggestLevel returns the lower-level suggestion for the saved settings, nil
// when none (learning.SuggestLowerLevel; one COUNT query when it can apply).
func (s *Service) SuggestLevel(ctx context.Context, userID string, st models.LearningSettings) (*LevelSuggestion, error) {
	now := s.now()
	if learning.SuggestLowerLevel(st.ReaderLevel, learning.LevelSuggestionMinEasier, st.LevelSuggestionDismissedAt, now) == "" {
		return nil, nil // 낮출 수준이 없거나 이번 주에 닫았다 — 셀 필요가 없다.
	}
	n, err := s.Store.EasierRequestsSince(ctx, userID, learning.LevelSuggestionSince(now, st.ReaderLevelChangedAt))
	if err != nil {
		return nil, err
	}
	to := learning.SuggestLowerLevel(st.ReaderLevel, n, st.LevelSuggestionDismissedAt, now)
	if to == "" {
		return nil, nil
	}
	return &LevelSuggestion{To: to, Reason: learning.LevelSuggestionEasierWhy}, nil
}

// DismissLevelSuggestion hides the suggestion for learning.LevelSuggestionWindow.
func (s *Service) DismissLevelSuggestion(ctx context.Context, userID string) error {
	return s.Store.DismissLevelSuggestion(ctx, userID, s.now())
}
