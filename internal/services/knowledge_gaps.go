package services

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/forgen-team/punisher/internal/prompts"
)

// ContextKindKnowledgeGap is the study request context kind witems rss-server
// sends from a knowledge card's '모름/충돌' (KNOWLEDGE-DESIGN §4.7):
//
//	{"kind":"knowledge_gap","gaps":[{"type":"unknown"|"conflict","text":"…","sides":["A","B"],"grade":"data|case|opinion|promo|unknown"}]}
const ContextKindKnowledgeGap = "knowledge_gap"

// Per-part caps so a conflict line keeps both sides under prompts.MaxKnowledgeGapRunes.
const (
	maxGapTextRunes = 160
	maxGapSideRunes = 60
)

type knowledgeGapContext struct {
	Kind string         `json:"kind"`
	Gaps []knowledgeGap `json:"gaps"`
}

type knowledgeGap struct {
	Type  string   `json:"type"`
	Text  string   `json:"text"`
	Sides []string `json:"sides"`
	Grade string   `json:"grade"`
}

// gapGrades are the evidence grades worth telling the lecture ("unknown" adds nothing).
var gapGrades = map[string]string{"data": "데이터", "case": "사례", "opinion": "의견", "promo": "홍보"}

// KnowledgeGapsFromContext turns a study request context of kind knowledge_gap
// into prompt lines ("모름: …", "충돌: … — ① … ② …"). Any other kind, broken
// JSON or unusable gap yields nil (the request keeps its old behavior).
func KnowledgeGapsFromContext(raw json.RawMessage) []string {
	var c knowledgeGapContext
	if len(raw) == 0 || json.Unmarshal(raw, &c) != nil || c.Kind != ContextKindKnowledgeGap {
		return nil
	}
	var out []string
	for _, g := range c.Gaps {
		if len(out) == prompts.MaxKnowledgeGaps {
			break
		}
		if line := gapLine(g); line != "" {
			out = append(out, line)
		}
	}
	return out
}

func gapLine(g knowledgeGap) string {
	text := cleanGapPart(g.Text, maxGapTextRunes)
	if text == "" {
		return ""
	}
	var line string
	switch g.Type {
	case "unknown":
		line = "모름: " + text
	case "conflict":
		line = "충돌: " + text
		var sides []string
		for _, s := range g.Sides {
			if s = cleanGapPart(s, maxGapSideRunes); s != "" && len(sides) < 2 {
				sides = append(sides, s)
			}
		}
		if len(sides) == 2 {
			line += " — ① " + sides[0] + " ② " + sides[1]
		}
	default:
		return ""
	}
	if name, ok := gapGrades[g.Grade]; ok {
		line += " (근거 등급: " + name + ")"
	}
	return line
}

// cleanGapPart collapses whitespace and cuts s to max runes.
func cleanGapPart(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}

// studyRequestGaps loads the user's study request and returns its knowledge
// gaps. A missing request or one of another kind is not an error for the
// lecture: it is logged and the lecture starts without gaps.
func (s *Service) studyRequestGaps(ctx context.Context, userID, requestID string) []string {
	if requestID == "" {
		return nil
	}
	req, err := s.Store.GetStudyRequest(ctx, userID, requestID)
	if err != nil {
		logf("study request %s for lecture: %v", requestID, err)
		return nil
	}
	return KnowledgeGapsFromContext(req.Context)
}
