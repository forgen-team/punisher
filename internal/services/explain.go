package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/lecture"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
)

// Selection limits for "이 부분 설명".
const (
	MinSelectionRunes = 2
	MaxSelectionRunes = 1000
)

// ErrInvalidSelection: the selected passage is empty or too long.
var ErrInvalidSelection = fmt.Errorf("selection must be %d..%d characters", MinSelectionRunes, MaxSelectionRunes)

// NormalizeSelection trims the selected text, collapses runs of blank lines
// and validates its length (in runes, after trimming).
func NormalizeSelection(text string) (string, error) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	blank := false
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			if blank {
				continue
			}
			blank = true
		} else {
			blank = false
		}
		out = append(out, line)
	}
	norm := strings.TrimSpace(strings.Join(out, "\n"))
	n := utf8.RuneCountInString(norm)
	if n < MinSelectionRunes || n > MaxSelectionRunes {
		return "", ErrInvalidSelection
	}
	return norm, nil
}

// Explanation is the answer to a selected passage (not stored; usage is recorded).
type Explanation struct {
	Selection    string `json:"selection"`
	SectionTitle string `json:"sectionTitle"`
	Explanation  string `json:"explanation"`
	Model        string `json:"model"`
}

// ExplainSelection explains a passage the user selected in a lecture, in the
// context of its section, on the interactive gateway lane.
func (s *Service) ExplainSelection(ctx context.Context, userID string, lec models.Lecture, anchor, selection string) (Explanation, error) {
	selection, err := NormalizeSelection(selection)
	if err != nil {
		return Explanation{}, err
	}
	pc, lang, err := s.explainContext(ctx, userID, lec, anchor)
	if err != nil {
		return Explanation{}, err
	}
	if err := s.CheckBudget(ctx, userID, false); err != nil {
		return Explanation{}, err
	}
	apiKey, err := s.Keys.RequestKey(ctx, userID)
	if err != nil {
		return Explanation{}, err
	}
	text, model, err := s.explainWithKey(ctx, userID, apiKey, lec.ConceptID, pc, lang, prompts.ExplainSelection(pc, selection))
	if err != nil {
		return Explanation{}, err
	}
	return Explanation{Selection: selection, SectionTitle: pc.SectionTitle, Explanation: text, Model: model}, nil
}

// explainContext is the section context of a passage explanation; lang is the
// lecture's language (passage answers follow it, i18n §1.1).
func (s *Service) explainContext(ctx context.Context, userID string, lec models.Lecture, anchor string) (prompts.ExplainContext, string, error) {
	var content lecture.Content
	if err := json.Unmarshal(lec.Content, &content); err != nil {
		return prompts.ExplainContext{}, "", fmt.Errorf("decode lecture content: %w", err)
	}
	section := SectionForAnchor(content, anchor)
	pc := prompts.ExplainContext{
		ConceptID: lec.ConceptID, LectureTitle: lec.Title, SectionTitle: section.Title,
		SectionText: RenderSection(section), ExampleLanguage: s.exampleLanguageName(ctx, userID),
	}
	lang := learning.NormalizeLocale(lec.Language) // 구절 설명은 강의 언어(i18n §1.1)
	pc.Reader = s.readerFor(ctx, userID, lang)     // 설명 밀도는 지금 읽는 사용자의 수준(learner-level ①)
	if info, err := s.Store.ConceptInfo(ctx, userID, lec.ConceptID, lang); err == nil {
		pc.ConceptName = info.Name
	}
	return pc, lang, nil
}

// explainWithKey runs a passage-explanation prompt on the explain block and the
// short lane; returns the trimmed markdown and the model (usage is recorded).
func (s *Service) explainWithKey(ctx context.Context, userID, apiKey, conceptID string, pc prompts.ExplainContext, lang, user string) (string, string, error) {
	var u usage
	text, err := s.complete(ctx, apiKey, llm.MessagesRequest{
		Model: s.Cfg.LectureModel,
		System: prompts.WithOutputLanguage(s.UserPrompts(ctx, userID).Text(prompts.BlockExplainSystem,
			map[string]string{"concept_name": pc.ConceptName, "example_language": pc.ExampleLanguage}), lang),
		User:      user,
		MaxTokens: 2000, Timeout: 90 * time.Second, Priority: llm.ShortUnlessBackground(ctx),
	}, &u)
	text = strings.TrimSpace(text)
	if err == nil && text == "" {
		err = errors.New("model returned an empty explanation")
	}
	s.recordUsage(ctx, userID, "explain", conceptID, u, err)
	return text, u.model, err
}
