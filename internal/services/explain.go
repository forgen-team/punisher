package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/llm"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/prompts"
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
	var content lecture.Content
	if err := json.Unmarshal(lec.Content, &content); err != nil {
		return Explanation{}, fmt.Errorf("decode lecture content: %w", err)
	}
	section := SectionForAnchor(content, anchor)
	pc := prompts.ExplainContext{
		ConceptID: lec.ConceptID, LectureTitle: lec.Title, SectionTitle: section.Title,
		SectionText: RenderSection(section), ExampleLanguage: s.exampleLanguageName(ctx, userID),
	}
	if info, err := s.Store.ConceptInfo(ctx, lec.ConceptID); err == nil {
		pc.ConceptName = info.Name
	}
	if err := s.CheckBudget(ctx, userID, false); err != nil {
		return Explanation{}, err
	}
	apiKey, err := s.Keys.RequestKey(ctx, userID)
	if err != nil {
		return Explanation{}, err
	}
	var u usage
	text, err := s.complete(ctx, apiKey, llm.MessagesRequest{
		Model: s.Cfg.LectureModel,
		System: s.UserPrompts(ctx, userID).Text(prompts.BlockExplainSystem,
			map[string]string{"concept_name": pc.ConceptName, "example_language": pc.ExampleLanguage}),
		User:      prompts.ExplainSelection(pc, selection),
		MaxTokens: 2000, Timeout: 90 * time.Second, Priority: llm.PriorityInteractive,
	}, &u)
	text = strings.TrimSpace(text)
	if err == nil && text == "" {
		err = errors.New("model returned an empty explanation")
	}
	s.recordUsage(ctx, userID, "explain", lec.ConceptID, u, err)
	if err != nil {
		return Explanation{}, err
	}
	return Explanation{Selection: selection, SectionTitle: section.Title, Explanation: text, Model: u.model}, nil
}
