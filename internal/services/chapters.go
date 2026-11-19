package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/forgen-team/punisher/internal/chapter"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/llmjson"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
)

const (
	maxChapterPicks     = 5
	maxChapterFreeRunes = 500
	chapterSuggestions  = 5
)

// ErrInvalidChapterRequest: nothing picked, or the request is too large.
var ErrInvalidChapterRequest = errors.New("pick at least one suggestion or type a topic (max 5 picks, 500 chars)")

// ChapterDesignError: the model could not produce a valid chapter (problems listed).
type ChapterDesignError struct{ Problems []string }

func (e *ChapterDesignError) Error() string {
	return "chapter design failed validation: " + strings.Join(e.Problems, "; ")
}

// SuggestChapters ranks next-chapter suggestions from the user's proposed
// concepts, unfinished depth, RSS interests/trends (when available) and new
// areas. An RSS failure is logged and the suggestions go on without it.
func (s *Service) SuggestChapters(ctx context.Context, userID string) ([]learning.Suggestion, error) {
	proposed, deepen, covered, err := s.Store.ChapterSignals(ctx, userID)
	if err != nil {
		return nil, err
	}
	in := learning.SuggestionInput{Proposed: proposed, Deepen: deepen, NewAreas: learning.DefaultNewAreas, Covered: covered,
		Locale: s.Store.LocaleFor(ctx, userID)}
	if s.Signals != nil {
		sig, err := s.Signals.LearningSignals(ctx, userID)
		if err != nil {
			logf("next-chapter RSS signals for %s: %v (suggesting without RSS)", userID, err)
		} else {
			applySignals(&in, sig)
		}
	}
	return learning.RankSuggestions(in, chapterSuggestions), nil
}

func applySignals(in *learning.SuggestionInput, sig LearningSignals) {
	in.Interests = sig.Interests
	for _, t := range sig.Trending {
		in.Trending = append(in.Trending, learning.TrendSignal{Topic: t.Topic, Score: t.Z})
	}
	mentions := map[string]int{}
	for _, c := range sig.ExplanationConcepts {
		mentions[c.ConceptID] = c.Count
	}
	read := map[string]int{}
	for _, c := range sig.ReadConcepts {
		read[c.ConceptID] = c.Count
	}
	for i := range in.Deepen {
		in.Deepen[i].Mentions = mentions[in.Deepen[i].ConceptID]
		in.Deepen[i].ReadMentions = read[in.Deepen[i].ConceptID]
	}
	for _, c := range sig.ReadCandidates {
		if name, ok := learning.CleanReadTopicName(c.Name); ok {
			in.ReadTopics = append(in.ReadTopics, learning.ReadTopicSignal{Name: name, Count: c.Count})
		}
	}
}

// ChapterRequest is what the user picked on "다음엔 어떤 걸 공부해 보고 싶어요?".
type ChapterRequest struct {
	SuggestionIDs []string `json:"suggestionIds"`
	FreeText      string   `json:"freeText"`
}

// ChapterDraftResult is a validated, stored preview.
type ChapterDraftResult struct {
	DraftID string        `json:"draftId"`
	Draft   chapter.Draft `json:"draft"`
}

// ValidateChapterRequest normalizes the request.
func ValidateChapterRequest(req ChapterRequest) (ChapterRequest, error) {
	req.FreeText = strings.TrimSpace(req.FreeText)
	ids := []string{}
	seen := map[string]bool{}
	for _, id := range req.SuggestionIDs {
		if id = strings.TrimSpace(id); id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	req.SuggestionIDs = ids
	if (len(ids) == 0 && req.FreeText == "") || len(ids) > maxChapterPicks ||
		utf8.RuneCountInString(req.FreeText) > maxChapterFreeRunes {
		return req, ErrInvalidChapterRequest
	}
	return req, nil
}

// CreateChapterDraft designs a chapter with the gateway (interactive lane),
// validates it (one repair attempt with the problems fed back) and stores it
// for preview. Nothing is added to the map until ApplyChapterDraft.
func (s *Service) CreateChapterDraft(ctx context.Context, userID string, req ChapterRequest) (ChapterDraftResult, error) {
	req, err := ValidateChapterRequest(req)
	if err != nil {
		return ChapterDraftResult{}, err
	}
	lang := s.Store.LocaleFor(ctx, userID) // 새 챕터는 사용자 언어(i18n §1.2)
	pc, cctx, err := s.chapterPromptContext(ctx, userID, req, lang)
	if err != nil {
		return ChapterDraftResult{}, err
	}
	if err := s.CheckBudget(ctx, userID, false); err != nil {
		return ChapterDraftResult{}, err
	}
	apiKey, err := s.Keys.RequestKey(ctx, userID)
	if err != nil {
		return ChapterDraftResult{}, err
	}
	var u usage
	var draft chapter.Draft
	var problems []string
	for attempt := 0; attempt < 2; attempt++ {
		pc.Revisions = problems
		var text string
		text, err = s.complete(ctx, apiKey, llm.MessagesRequest{
			Model: s.Cfg.LectureModel, System: prompts.WithOutputLanguage(prompts.SystemChapter, lang), User: prompts.Chapter(pc),
			MaxTokens: 8000, Timeout: 4 * time.Minute, Priority: llm.PriorityInteractive,
		}, &u)
		if err != nil {
			break
		}
		var raw chapter.Draft
		if raw, err = llmjson.TryParseJSON[chapter.Draft](text); err != nil {
			problems = []string{"출력이 JSON 형식이 아닙니다: " + err.Error()}
			err = nil
			continue
		}
		if draft, problems = chapter.Validate(raw, cctx); len(problems) == 0 {
			break
		}
	}
	if err == nil && len(problems) > 0 {
		err = &ChapterDesignError{Problems: problems}
	}
	s.recordUsage(ctx, userID, "chapter", "", u, err)
	if err != nil {
		return ChapterDraftResult{}, err
	}
	id, err := s.Store.SaveChapterDraft(ctx, userID, req, draft, u.model)
	if err != nil {
		return ChapterDraftResult{}, err
	}
	return ChapterDraftResult{DraftID: id, Draft: draft}, nil
}

func (s *Service) chapterPromptContext(ctx context.Context, userID string, req ChapterRequest, lang string) (prompts.ChapterContext, chapter.Context, error) {
	pc := prompts.ChapterContext{FreeText: req.FreeText, Areas: curriculum.Areas}
	if len(req.SuggestionIDs) > 0 {
		suggestions, err := s.SuggestChapters(ctx, userID)
		if err != nil {
			return pc, chapter.Context{}, err
		}
		byID := map[string]learning.Suggestion{}
		for _, sg := range suggestions {
			byID[sg.ID] = sg
		}
		for _, id := range req.SuggestionIDs {
			if sg, ok := byID[id]; ok {
				pc.Picks = append(pc.Picks, prompts.ChapterPick{Title: sg.Title, Reason: sg.Reason, ConceptIDs: sg.ConceptIDs, Untrusted: sg.Kind == learning.SuggestReading})
			}
		}
		if len(pc.Picks) == 0 && req.FreeText == "" {
			return pc, chapter.Context{}, fmt.Errorf("%w: suggestions are no longer available", ErrInvalidChapterRequest)
		}
	}
	cctx, err := s.Store.ChapterContext(ctx, userID)
	if err != nil {
		return pc, cctx, err
	}
	cctx.Areas = curriculum.Areas
	catalog, err := s.Store.ConceptCatalog(ctx, userID, lang)
	if err != nil {
		return pc, cctx, err
	}
	for _, c := range catalog {
		ref := prompts.ConceptRef{ID: c.ID, Name: c.Name}
		if name, own := cctx.OwnProposed[c.ID]; own {
			pc.Proposed = append(pc.Proposed, prompts.ConceptRef{ID: c.ID, Name: name})
			continue
		}
		if learning.IsDone(c.State) {
			pc.Learned = append(pc.Learned, ref)
		}
		pc.Catalog = append(pc.Catalog, prompts.CatalogRef{ID: c.ID, Name: c.Name, Area: c.Area, Learned: learning.IsDone(c.State)})
	}
	return pc, cctx, nil
}

// ApplyChapterDraft re-validates the stored draft against the user's current
// map and opens it as a chapter.
func (s *Service) ApplyChapterDraft(ctx context.Context, userID, draftID string) (string, error) {
	d, input, applied, err := s.Store.ChapterDraft(ctx, userID, draftID)
	if err != nil {
		return "", err
	}
	if applied {
		return "", store.ErrDraftApplied
	}
	cctx, err := s.Store.ChapterContext(ctx, userID)
	if err != nil {
		return "", err
	}
	cctx.Areas = curriculum.Areas
	d, problems := chapter.Validate(d, cctx)
	if len(problems) > 0 {
		return "", &ChapterDesignError{Problems: problems}
	}
	return s.Store.ApplyChapter(ctx, userID, draftID, d, input)
}
