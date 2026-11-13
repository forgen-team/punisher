package check

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Item rubric bounds (design §4.1).
const (
	MinMust = 1
	MaxMust = 4
)

// SetContext is what an item set is validated against: the lecture.
type SetContext struct {
	// ObjectiveIDs: the lecture's objectives, in order.
	ObjectiveIDs []string
	// Anchors: section/block anchors of the lecture (rubric.anchor must be one).
	Anchors map[string]bool
}

var formats = map[string]bool{FormatRecall: true, FormatPredict: true, FormatApply: true, FormatIntegrative: true}

// ValidateItems lists the problems of a generated set (empty = usable): per
// objective one check and one review item, plus one integrative check item.
func ValidateItems(set ItemSet, sc SetContext) []string {
	var problems []string
	inLecture := map[string]bool{}
	for _, id := range sc.ObjectiveIDs {
		inLecture[id] = true
	}
	counts := map[string]int{}
	integrative := 0
	for i, it := range set.Items {
		label := fmt.Sprintf("items[%d]", i)
		problems = append(problems, validateItem(it, sc, label)...)
		switch {
		case it.Integrative():
			integrative++
		case !inLecture[it.ObjectiveID]:
			problems = append(problems, fmt.Sprintf("%s: objective_id %q 는 이 강의의 목표가 아닙니다", label, it.ObjectiveID))
		default:
			counts[it.ObjectiveID+"|"+it.Purpose]++
		}
	}
	for _, id := range sc.ObjectiveIDs {
		for _, p := range []string{PurposeCheck, PurposeReview} {
			if n := counts[id+"|"+p]; n != 1 {
				problems = append(problems, fmt.Sprintf("목표 %s 의 %s 문항은 정확히 1개여야 합니다(현재 %d개)", id, p, n))
			}
		}
	}
	if integrative != 1 {
		problems = append(problems, fmt.Sprintf("통합 문항(objective_id 빈 값, format integrative)은 정확히 1개여야 합니다(현재 %d개)", integrative))
	}
	return problems
}

func validateItem(it Item, sc SetContext, label string) []string {
	var p []string
	if it.Purpose != PurposeCheck && it.Purpose != PurposeReview {
		p = append(p, label+": purpose 는 check 또는 review 입니다")
	}
	if !formats[it.Format] {
		p = append(p, label+": format 은 recall|predict|apply|integrative 중 하나입니다")
	}
	if it.Integrative() != (it.Format == FormatIntegrative) {
		p = append(p, label+": 통합 문항만 objective_id 를 비우고 format integrative 를 씁니다")
	}
	if it.Integrative() && it.Purpose != PurposeCheck {
		p = append(p, label+": 통합 문항의 purpose 는 check 입니다")
	}
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
	if !sc.Anchors[it.Rubric.Anchor] {
		p = append(p, fmt.Sprintf("%s: rubric.anchor %q 는 강의 본문의 앵커가 아닙니다", label, it.Rubric.Anchor))
	}
	if len(it.Material) > 0 && !isJSONObject(it.Material) {
		p = append(p, label+": material 은 JSON 객체입니다")
	}
	return p
}

// NormalizeItems prepares a validated set for storage: trimmed text, an object
// material, and covers = the lecture's objectives on the integrative item
// (code decides the credit range, not the model — design §4.2 N9).
func NormalizeItems(set ItemSet, objectiveIDs []string) []Item {
	out := make([]Item, 0, len(set.Items))
	for _, it := range set.Items {
		it.Prompt = strings.TrimSpace(it.Prompt)
		it.Rubric.Must = nonEmpty(it.Rubric.Must)
		it.Rubric.Nice = nonEmpty(it.Rubric.Nice)
		it.Rubric.Misconceptions = nonEmpty(it.Rubric.Misconceptions)
		it.Rubric.Answer = strings.TrimSpace(it.Rubric.Answer)
		it.Rubric.Covers = []string{}
		if it.Integrative() {
			it.Rubric.Covers = append([]string{}, objectiveIDs...)
		}
		if len(it.Material) == 0 || !isJSONObject(it.Material) {
			it.Material = json.RawMessage(`{}`)
		}
		out = append(out, it)
	}
	return out
}

func nonEmpty(list []string) []string {
	out := []string{}
	for _, s := range list {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func isJSONObject(raw json.RawMessage) bool {
	t := bytes.TrimSpace(raw)
	return len(t) > 0 && t[0] == '{' && json.Valid(t)
}

// ValidateVariants lists the problems of a generated review-variant set (empty
// = usable): exactly one review item for each requested objective, nothing else.
func ValidateVariants(set ItemSet, sc SetContext, objectiveIDs []string) []string {
	var problems []string
	want := map[string]bool{}
	for _, id := range objectiveIDs {
		want[id] = true
	}
	counts := map[string]int{}
	for i, it := range set.Items {
		label := fmt.Sprintf("items[%d]", i)
		problems = append(problems, validateItem(it, sc, label)...)
		switch {
		case it.Purpose != PurposeReview:
			problems = append(problems, label+": 변형 문항의 purpose 는 review 입니다")
		case !want[it.ObjectiveID]:
			problems = append(problems, fmt.Sprintf("%s: objective_id %q 는 요청한 목표가 아닙니다", label, it.ObjectiveID))
		default:
			counts[it.ObjectiveID]++
		}
	}
	for _, id := range objectiveIDs {
		if n := counts[id]; n != 1 {
			problems = append(problems, fmt.Sprintf("목표 %s 의 review 문항은 정확히 1개여야 합니다(현재 %d개)", id, n))
		}
	}
	return problems
}
