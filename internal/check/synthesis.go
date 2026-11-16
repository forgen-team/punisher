package check

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PurposeSynthesis: the weekly item set mixing several checked concepts (v3 design §5).
// Its items are integrative (objective_id empty) and cover objectives of 2+ concepts.
const PurposeSynthesis = "synthesis"

// SynthesisItemCount: items of one synthesis session (scene diagnosis, prescription comparison).
const SynthesisItemCount = 2

// SynthesisContext is what a synthesis set is validated against.
type SynthesisContext struct {
	// ObjectiveIDs: the picked objectives; ConceptOf maps each to its concept.
	ObjectiveIDs []string
	ConceptOf    map[string]string
}

// ValidateSynthesis lists the problems of a generated synthesis set: exactly
// SynthesisItemCount integrative items, each covering picked objectives of at
// least two concepts, with a one-question prompt and a usable rubric.
func ValidateSynthesis(set ItemSet, sc SynthesisContext) []string {
	var p []string
	if len(set.Items) != SynthesisItemCount {
		p = append(p, fmt.Sprintf("종합 문항은 정확히 %d개여야 합니다(현재 %d개)", SynthesisItemCount, len(set.Items)))
	}
	for i, it := range set.Items {
		label := fmt.Sprintf("items[%d]", i)
		// objective_id·format·purpose 는 코드가 정한다(NormalizeSynthesis) — 모델이 채워도 탓하지 않는다.
		if strings.TrimSpace(it.Prompt) == "" {
			p = append(p, label+": prompt 가 비었습니다")
		} else if CountQuestions(it.Prompt) > 1 {
			p = append(p, label+": prompt 에는 물음이 하나만 있어야 합니다")
		}
		if n := len(nonEmpty(it.Rubric.Must)); n < MinMust || n > MaxMust {
			p = append(p, fmt.Sprintf("%s: rubric.must 는 %d~%d개입니다(현재 %d개)", label, MinMust, MaxMust, n))
		}
		if strings.TrimSpace(it.Rubric.Answer) == "" {
			p = append(p, label+": rubric.answer 가 비었습니다")
		} else if Leaks(it.Prompt, it.Rubric.Answer) {
			p = append(p, label+": prompt 가 모범답을 그대로 드러냅니다")
		}
		concepts := map[string]bool{}
		for _, id := range it.Rubric.Covers {
			c, ok := sc.ConceptOf[id]
			if !ok {
				p = append(p, fmt.Sprintf("%s: rubric.covers 의 %q 는 고른 목표가 아닙니다", label, id))
				continue
			}
			concepts[c] = true
		}
		if len(concepts) < 2 {
			p = append(p, label+": rubric.covers 는 서로 다른 개념 2개 이상의 목표를 담아야 합니다")
		}
		if len(it.Material) > 0 && !isJSONObject(it.Material) {
			p = append(p, label+": material 은 JSON 객체입니다")
		}
	}
	return p
}

// NormalizeSynthesis prepares a validated set for storage: purpose synthesis,
// trimmed text, covers deduplicated in the plan's order, an object material.
func NormalizeSynthesis(set ItemSet, objectiveIDs []string) []Item {
	out := make([]Item, 0, len(set.Items))
	for _, it := range set.Items {
		it.Purpose, it.ObjectiveID, it.Format = PurposeSynthesis, "", FormatIntegrative
		it.Prompt = strings.TrimSpace(it.Prompt)
		it.Rubric.Must = nonEmpty(it.Rubric.Must)
		it.Rubric.Nice = nonEmpty(it.Rubric.Nice)
		it.Rubric.Misconceptions = nonEmpty(it.Rubric.Misconceptions)
		it.Rubric.Answer = strings.TrimSpace(it.Rubric.Answer)
		it.Rubric.Anchor = ""
		want := map[string]bool{}
		for _, id := range it.Rubric.Covers {
			want[id] = true
		}
		it.Rubric.Covers = []string{}
		for _, id := range objectiveIDs {
			if want[id] {
				it.Rubric.Covers = append(it.Rubric.Covers, id)
			}
		}
		if len(it.Material) == 0 || !isJSONObject(it.Material) {
			it.Material = json.RawMessage(`{}`)
		}
		out = append(out, it)
	}
	return out
}
