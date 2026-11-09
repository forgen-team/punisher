package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/llm"
	"github.com/wooo-jin/punisher/internal/llmjson"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/prompts"
)

// Gloss returns the cached short gloss of a concept, generating it with the
// fast model on a cache miss. cached reports whether no LLM call was made.
// The shared cache holds default-prompt glosses only; a user who edited the
// gloss block gets a private cache keyed by the block hash.
func (s *Service) Gloss(ctx context.Context, userID, conceptID string) (gloss string, cached bool, err error) {
	set := s.UserPrompts(ctx, userID)
	custom := set.Overridden(prompts.BlockGlossSystem)
	hash := set.Hash(prompts.BlockGlossSystem)
	if custom {
		gloss, err = s.Store.UserGloss(ctx, userID, conceptID, hash)
	} else {
		gloss, err = s.Store.GetGloss(ctx, conceptID)
	}
	if err != nil || gloss != "" {
		return gloss, gloss != "", err
	}
	concept, err := s.Store.ConceptInfo(ctx, conceptID)
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
		Model: s.Cfg.FastModel, System: set.Text(prompts.BlockGlossSystem, map[string]string{"concept_name": concept.Name}),
		User: prompts.Gloss(concept.Name, concept.Area, concept.Summary), MaxTokens: 400, Timeout: 30 * time.Second,
	}, &u)
	s.recordUsage(ctx, userID, "gloss", conceptID, u, err)
	if err != nil {
		return "", false, err
	}
	gloss = strings.TrimSpace(text)
	if custom {
		err = s.Store.SaveUserGloss(ctx, userID, conceptID, hash, gloss, u.model)
	} else {
		err = s.Store.SaveGloss(ctx, conceptID, gloss, u.model)
	}
	if err != nil {
		return "", false, err
	}
	return gloss, false, nil
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
		return s.Store.FailQuestion(ctx, q.ID, "", err.Error())
	}
	return s.answer(ctx, userID, q, qc.known, prompts.Question(qc.prompt, question))
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
		Question: strings.TrimSpace("다른 방식으로 다시 설명해 주세요. " + comment),
	})
	if err != nil {
		return q, err
	}
	qc, err := s.qaContext(ctx, userID, lec, anchor, false)
	if err != nil {
		return s.Store.FailQuestion(ctx, q.ID, "", err.Error())
	}
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		if _, err := s.answerWithKey(bg, userID, apiKey, q, qc.known, prompts.AltExplanation(qc.prompt, comment)); err != nil {
			logf("alt explanation %s: %v", q.ID, err)
		}
	}()
	return q, nil
}

func (s *Service) answer(ctx context.Context, userID string, q models.Question, known map[string]bool, prompt string) (models.Question, error) {
	if err := s.CheckBudget(ctx, userID, false); err != nil {
		return s.Store.FailQuestion(ctx, q.ID, "", err.Error())
	}
	apiKey, err := s.Keys.RequestKey(ctx, userID)
	if err != nil {
		return s.Store.FailQuestion(ctx, q.ID, "", err.Error())
	}
	return s.answerWithKey(ctx, userID, apiKey, q, known, prompt)
}

func (s *Service) answerWithKey(ctx context.Context, userID, apiKey string, q models.Question, known map[string]bool, prompt string) (models.Question, error) {
	var u usage
	text, err := s.complete(ctx, apiKey, llm.MessagesRequest{
		Model: s.Cfg.LectureModel, System: s.UserPrompts(ctx, userID).Text(prompts.BlockQASystem, s.interactionVars(ctx, userID, q.ConceptID)),
		User: prompt, MaxTokens: 3000, Timeout: 2 * time.Minute,
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
		return s.Store.FailQuestion(ctx, q.ID, u.model, err.Error())
	}
	linked := []string{}
	for _, id := range out.LinkedConceptIDs {
		if known[id] {
			linked = append(linked, id)
		}
	}
	proposed, _ := json.Marshal(out.ProposedConcepts)
	return s.Store.AnswerQuestion(ctx, q.ID, out.Answer, u.model, linked, proposed)
}

type qaCtx struct {
	prompt prompts.QAContext
	known  map[string]bool
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
	if info, err := s.Store.ConceptInfo(ctx, lec.ConceptID); err == nil {
		pc.ConceptName = info.Name
	}
	related, err := s.Store.RelatedConcepts(ctx, userID, lec.ConceptID)
	if err != nil {
		return qaCtx{}, err
	}
	known, err := s.Store.KnownConceptIDs(ctx)
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
	return qaCtx{prompt: pc, known: known}, nil
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
		default:
			b.WriteString(blk.Text + "\n")
		}
	}
	return b.String()
}
