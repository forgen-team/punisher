package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/forgen-team/punisher/internal/glossary"
	"github.com/forgen-team/punisher/internal/store"
	"strings"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/lecture"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/llmjson"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
)

// Gloss returns the cached short gloss of a concept, generating it with the
// fast model on a cache miss. cached reports whether no LLM call was made.
// The shared cache holds default-prompt Korean glosses only; a user who edited
// the gloss block, or reads in English, gets a private cache keyed by the block
// hash plus the language (userGlossKey, i18n §9).
func (s *Service) Gloss(ctx context.Context, userID, conceptID string) (gloss string, cached bool, err error) {
	lang := s.Store.LocaleFor(ctx, userID) // 팝오버는 화면(사용자) 언어(i18n §9)
	if glossary.IsTermID(conceptID) {      // 기본 용어: 사전에만 있다(생성하지 않음).
		t, ok := s.Glossary.TermIn(conceptID, lang)
		if !ok {
			return "", false, store.ErrNotFound
		}
		return t.Gloss, true, nil
	}
	set := s.UserPrompts(ctx, userID)
	custom := set.Overridden(prompts.BlockGlossSystem)
	if shipped := s.Glossary.ConceptGlossIn(conceptID, lang); shipped != "" && !custom {
		return shipped, true, nil // 사전 설명이 있으면 늘 같은 설명(프롬프트를 고친 사용자만 따로 만든다).
	}
	private := custom || lang == learning.LocaleEN
	hash := userGlossKey(set.Hash(prompts.BlockGlossSystem), lang)
	if private {
		gloss, err = s.Store.UserGloss(ctx, userID, conceptID, hash)
	} else {
		gloss, err = s.Store.GetGloss(ctx, conceptID)
	}
	if err != nil || gloss != "" {
		return gloss, gloss != "", err
	}
	concept, err := s.Store.ConceptInfo(ctx, userID, conceptID, lang)
	if err != nil {
		return "", false, err
	}
	if !s.AllowUser(userID) {
		return "", false, ErrNotOwner
	}
	if err := s.CheckBudget(ctx, userID, false); err != nil {
		return "", false, err
	}
	apiKey, err := s.Keys.RequestKey(ctx, userID)
	if err != nil {
		return "", false, err
	}
	var u usage
	text, err := s.complete(ctx, apiKey, llm.MessagesRequest{
		Model: s.Cfg.FastModel, System: prompts.WithOutputLanguage(set.Text(prompts.BlockGlossSystem, map[string]string{"concept_name": concept.Name}), lang),
		User: prompts.Gloss(concept.Name, concept.Area, concept.Summary), MaxTokens: 400, Timeout: 30 * time.Second,
		Priority: llm.ShortUnlessBackground(ctx),
	}, &u)
	s.recordUsage(ctx, userID, "gloss", conceptID, u, err)
	if err != nil {
		return "", false, err
	}
	gloss = strings.TrimSpace(text)
	if private {
		err = s.Store.SaveUserGloss(ctx, userID, conceptID, hash, gloss, u.model)
	} else {
		err = s.Store.SaveGloss(ctx, conceptID, gloss, u.model)
	}
	if err != nil {
		return "", false, err
	}
	return gloss, false, nil
}

// userGlossKey is the user_term_glosses.prompt_hash of a gloss: the gloss block
// hash ("" = default prompt) for Korean (keys made before i18n), and
// "<hash>|en" — "default|en" for the default prompt — for English.
func userGlossKey(hash, lang string) string {
	if learning.NormalizeLocale(lang) == learning.LocaleKO {
		return hash
	}
	if hash == "" {
		hash = "default"
	}
	return hash + learning.LanguageSuffix(lang)
}

// qaOutput is the Q&A / alt-explanation JSON.
type qaOutput struct {
	Answer           string            `json:"answer"`
	LinkedConceptIDs []string          `json:"linked_concept_ids"`
	ProposedConcepts []json.RawMessage `json:"proposed_concepts"`
}

// Ask answers a contextual question synchronously and stores the thread entry.
// LLM failure is stored as status=failed (never as an answer).
func (s *Service) Ask(ctx context.Context, userID string, lec models.Lecture, anchor, question string) (models.Question, error) {
	q, err := s.Store.CreateQuestion(ctx, userID, models.Question{
		LectureID: lec.ID, ConceptID: lec.ConceptID, Anchor: anchor, Kind: "question", Question: question,
	})
	if err != nil {
		return q, err
	}
	qc, err := s.qaContext(ctx, userID, lec, anchor, true)
	if err != nil {
		return s.Store.FailQuestion(ctx, q.ID, "", questionFailText(err))
	}
	return s.answer(ctx, userID, q, qc, prompts.Question(qc.prompt, question))
}

// RequestAltExplanation creates an alt_explanation entry and answers it in
// the background (key resolved now, while the session is available).
func (s *Service) RequestAltExplanation(ctx context.Context, userID string, lec models.Lecture, anchor, comment string) (models.Question, error) {
	if err := s.CheckBudget(ctx, userID, false); err != nil {
		return models.Question{}, err
	}
	apiKey, err := s.Keys.RequestKey(ctx, userID)
	if err != nil {
		return models.Question{}, err
	}
	q, err := s.Store.CreateQuestion(ctx, userID, models.Question{
		LectureID: lec.ID, ConceptID: lec.ConceptID, Anchor: anchor, Kind: "alt_explanation",
		Question: strings.TrimSpace(altExplanationRequest(lec.Language) + comment),
	})
	if err != nil {
		return q, err
	}
	qc, err := s.qaContext(ctx, userID, lec, anchor, false)
	if err != nil {
		return s.Store.FailQuestion(ctx, q.ID, "", questionFailText(err))
	}
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		defer recoverPanic("alt explanation", q.ID, func(err error) {
			if _, ferr := s.Store.FailQuestion(context.Background(), q.ID, "", questionFailText(err)); ferr != nil {
				logf("alt explanation %s: fail after panic: %v", q.ID, ferr)
			}
		})
		if _, err := s.answerWithKey(bg, userID, apiKey, q, qc, prompts.AltExplanation(qc.prompt, comment)); err != nil {
			logf("alt explanation %s: %v", q.ID, err)
		}
	}()
	return q, nil
}

func (s *Service) answer(ctx context.Context, userID string, q models.Question, qc qaCtx, prompt string) (models.Question, error) {
	if err := s.CheckBudget(ctx, userID, false); err != nil {
		return s.Store.FailQuestion(ctx, q.ID, "", questionFailText(err))
	}
	apiKey, err := s.Keys.RequestKey(ctx, userID)
	if err != nil {
		return s.Store.FailQuestion(ctx, q.ID, "", questionFailText(err))
	}
	return s.answerWithKey(ctx, userID, apiKey, q, qc, prompt)
}

func (s *Service) answerWithKey(ctx context.Context, userID, apiKey string, q models.Question, qc qaCtx, prompt string) (models.Question, error) {
	var u usage
	system := s.UserPrompts(ctx, userID).Text(prompts.BlockQASystem, s.interactionVars(ctx, userID, q.ConceptID, qc.lang))
	text, err := s.complete(ctx, apiKey, llm.MessagesRequest{
		Model: s.Cfg.LectureModel, System: prompts.WithOutputLanguage(system, qc.lang),
		User: prompt, MaxTokens: 3000, Timeout: 2 * time.Minute, Priority: llm.ShortUnlessBackground(ctx),
	}, &u)
	var out qaOutput
	if err == nil {
		out, err = llmjson.TryParseJSON[qaOutput](text)
		if err == nil && strings.TrimSpace(out.Answer) == "" {
			err = fmt.Errorf("model returned an empty answer")
		}
	}
	s.recordUsage(ctx, userID, "question", q.ConceptID, u, err)
	if err != nil {
		return s.Store.FailQuestion(ctx, q.ID, u.model, questionFailText(err))
	}
	linked := []string{}
	for _, id := range out.LinkedConceptIDs {
		if qc.known[id] {
			linked = append(linked, id)
		}
	}
	proposed, _ := json.Marshal(out.ProposedConcepts)
	return s.Store.AnswerQuestion(ctx, q.ID, out.Answer, u.model, linked, proposed)
}

type qaCtx struct {
	prompt prompts.QAContext
	known  map[string]bool
	lang   string // the lecture's language: answers follow it (i18n §1.1)
}

func (s *Service) qaContext(ctx context.Context, userID string, lec models.Lecture, anchor string, withHistory bool) (qaCtx, error) {
	var content lecture.Content
	if err := json.Unmarshal(lec.Content, &content); err != nil {
		return qaCtx{}, fmt.Errorf("decode lecture content: %w", err)
	}
	section := SectionForAnchor(content, anchor)
	pc := prompts.QAContext{
		ConceptID: lec.ConceptID, LectureTitle: lec.Title,
		SectionTitle: section.Title, SectionText: RenderSection(section),
		ExampleLanguage: s.exampleLanguageName(ctx, userID),
	}
	lang := learning.NormalizeLocale(lec.Language)
	if info, err := s.Store.ConceptInfo(ctx, userID, lec.ConceptID, lang); err == nil {
		pc.ConceptName = info.Name
	}
	related, err := s.Store.RelatedConcepts(ctx, userID, lec.ConceptID, lang)
	if err != nil {
		return qaCtx{}, err
	}
	known, err := s.Store.KnownConceptIDs(ctx, userID)
	if err != nil {
		return qaCtx{}, err
	}
	for _, r := range related {
		ref := prompts.ConceptRef{ID: r.ID, Name: r.Name}
		pc.Linkable = append(pc.Linkable, ref)
		if learning.IsDone(r.State) {
			pc.LearnedBefore = append(pc.LearnedBefore, ref)
		}
	}
	if pc.Misconceptions, err = s.promptMisconceptions(ctx, userID, lec.ConceptID); err != nil {
		return qaCtx{}, err
	}
	if withHistory {
		prev, err := s.Store.ListQuestions(ctx, userID, lec.ID)
		if err != nil {
			return qaCtx{}, err
		}
		for _, p := range prev {
			if p.Status == "answered" && p.Kind == "question" {
				pc.History = append(pc.History, prompts.QATurn{Question: p.Question, Answer: p.Answer})
			}
		}
		if len(pc.History) > 5 {
			pc.History = pc.History[len(pc.History)-5:]
		}
	}
	return qaCtx{prompt: pc, known: known, lang: lang}, nil
}

// SectionForAnchor finds the section of a section/block anchor
// ("principles" or "principles-3"); defaults to the first section.
func SectionForAnchor(c lecture.Content, anchor string) lecture.Section {
	for _, s := range c.Sections {
		if anchor == s.Anchor || strings.HasPrefix(anchor, s.Anchor+"-") {
			return s
		}
	}
	if len(c.Sections) > 0 {
		return c.Sections[0]
	}
	return lecture.Section{}
}

// RenderSection flattens a section to plain text for prompts.
func RenderSection(s lecture.Section) string {
	var b strings.Builder
	for _, blk := range s.Blocks {
		switch blk.Type {
		case lecture.BlockCode, lecture.BlockLog, lecture.BlockMermaid:
			lang := blk.Lang
			if blk.Type == lecture.BlockMermaid {
				lang = "mermaid"
			}
			fmt.Fprintf(&b, "```%s\n%s\n```\n", lang, blk.Text)
		case lecture.BlockTerm:
			fmt.Fprintf(&b, "[용어 %s(%s)] %s\n", blk.Term, blk.ConceptID, blk.Text)
		case lecture.BlockMisconception:
			fmt.Fprintf(&b, "[흔한 오해] %s\n", blk.Text)
		case lecture.BlockSources:
			for _, src := range blk.Sources {
				fmt.Fprintf(&b, "[출처] %s %s\n", src.Title, src.URL)
			}
		case lecture.BlockTrace, lecture.BlockChart, lecture.BlockDiff:
			fmt.Fprintf(&b, "[%s %s] %s\n", sectionBlockLabels[blk.Type], blk.Lang, blk.Data)
		case lecture.BlockKeypoint, lecture.BlockPause, lecture.BlockExercise, lecture.BlockAnswer, lecture.BlockSummary:
			fmt.Fprintf(&b, "[%s] %s\n", sectionBlockLabels[blk.Type], blk.Text)
		default:
			b.WriteString(blk.Text + "\n")
		}
	}
	return b.String()
}

// sectionBlockLabels name the material blocks in prompt text.
var sectionBlockLabels = map[string]string{
	lecture.BlockTrace: "손계산 표", lecture.BlockChart: "차트", lecture.BlockDiff: "전후 비교",
	lecture.BlockKeypoint: "핵심", lecture.BlockPause: "예측 질문", lecture.BlockExercise: "직접 해 보기",
	lecture.BlockAnswer: "풀이", lecture.BlockSummary: "한 장 요약",
}

// altExplanationRequest is the stored question text of '다른 방식으로 설명' in the lecture's language.
func altExplanationRequest(lang string) string {
	if learning.NormalizeLocale(lang) == learning.LocaleEN {
		return "Please explain this another way. "
	}
	return "다른 방식으로 다시 설명해 주세요. "
}

// questionFailText is the stored error of a failed question / alt explanation
// (shown under it): the reason text of a Claude failure, which the web
// translates, else the error itself (logged details for budget/key errors).
func questionFailText(err error) string {
	if msg := DescribeLLMError(err); msg != "" {
		return msg
	}
	return err.Error()
}
