package check

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Diagnostic set bounds (design §5 P3).
const (
	MinPretestItems = 2
	MaxPretestItems = 3
	MinChoices      = 3
	MaxChoices      = 4
	// choiceLengthRatio: a correct option this many times longer than every
	// other option gives itself away.
	choiceLengthRatio = 2
)

// PretestCount is how many prediction questions a lecture start gets: one per
// objective, between MinPretestItems and MaxPretestItems.
func PretestCount(objectives int) int {
	return min(max(objectives, MinPretestItems), MaxPretestItems)
}

// DiagnosticContext is what a diagnostic set is validated against.
type DiagnosticContext struct {
	Purpose string
	// ObjectiveIDs: the objectives to cover. pretest: every one at least once
	// (PretestCount items in all); known_verify · placement: each exactly once.
	ObjectiveIDs []string
}

var diagnosticFormats = map[string]map[string]bool{
	PurposePretest:     {FormatChoice: true, FormatPredict: true},
	PurposeKnownVerify: {FormatRecall: true, FormatPredict: true, FormatApply: true},
	PurposePlacement:   {FormatChoice: true},
}

var diagnosticFormatHelp = map[string]string{
	PurposePretest:     "choice 또는 predict(한 줄 예측)",
	PurposeKnownVerify: "recall|predict|apply",
	PurposePlacement:   "choice",
}

// ValidateDiagnostic lists the problems of a generated pretest, known
// verification or placement set (empty = usable).
func ValidateDiagnostic(set ItemSet, dc DiagnosticContext) []string {
	formats, ok := diagnosticFormats[dc.Purpose]
	if !ok {
		return []string{fmt.Sprintf("알 수 없는 진단 종류 %q", dc.Purpose)}
	}
	var problems []string
	want := map[string]bool{}
	for _, id := range dc.ObjectiveIDs {
		want[id] = true
	}
	counts := map[string]int{}
	for i, it := range set.Items {
		label := fmt.Sprintf("items[%d]", i)
		if it.Purpose != dc.Purpose {
			problems = append(problems, fmt.Sprintf("%s: purpose 는 %s 입니다", label, dc.Purpose))
		}
		if !formats[it.Format] {
			problems = append(problems, fmt.Sprintf("%s: format 은 %s 입니다", label, diagnosticFormatHelp[dc.Purpose]))
		}
		if !want[it.ObjectiveID] {
			problems = append(problems, fmt.Sprintf("%s: objective_id %q 는 요청한 목표가 아닙니다", label, it.ObjectiveID))
		}
		counts[it.ObjectiveID]++
		problems = append(problems, validateDiagnosticItem(it, dc.Purpose, label)...)
	}
	if dc.Purpose == PurposePretest {
		if n := PretestCount(len(dc.ObjectiveIDs)); len(set.Items) != n {
			problems = append(problems, fmt.Sprintf("문항은 정확히 %d개여야 합니다(현재 %d개)", n, len(set.Items)))
		}
	}
	for _, id := range dc.ObjectiveIDs {
		n := counts[id]
		switch {
		case dc.Purpose == PurposePretest && n == 0:
			problems = append(problems, fmt.Sprintf("목표 %s 의 문항이 없습니다", id))
		case dc.Purpose != PurposePretest && n != 1:
			problems = append(problems, fmt.Sprintf("목표 %s 의 문항은 정확히 1개여야 합니다(현재 %d개)", id, n))
		}
	}
	return problems
}

func validateDiagnosticItem(it Item, purpose, label string) []string {
	var p []string
	if strings.TrimSpace(it.Prompt) == "" {
		p = append(p, label+": prompt 가 비었습니다")
	} else if CountQuestions(it.Prompt) > 1 {
		p = append(p, label+": prompt 에는 물음이 하나만 있어야 합니다")
	}
	if len(it.Material) > 0 && !isJSONObject(it.Material) {
		p = append(p, label+": material 은 JSON 객체입니다")
	}
	if it.Format == FormatChoice {
		return append(p, validateChoice(it, label)...)
	}
	if purpose == PurposeKnownVerify {
		if n := len(nonEmpty(it.Rubric.Must)); n < MinMust || n > MaxMust {
			p = append(p, fmt.Sprintf("%s: rubric.must 는 %d~%d개입니다(현재 %d개)", label, MinMust, MaxMust, n))
		}
	}
	if strings.TrimSpace(it.Rubric.Answer) == "" {
		p = append(p, label+": rubric.answer 가 비었습니다")
	} else if Leaks(it.Prompt, it.Rubric.Answer) {
		p = append(p, label+": prompt 가 모범답을 그대로 드러냅니다")
	}
	return p
}

// Choices are the options of a choice item (material.choices, trimmed).
func Choices(material json.RawMessage) []string {
	var m struct {
		Choices []string `json:"choices"`
	}
	if len(material) == 0 || json.Unmarshal(material, &m) != nil {
		return nil
	}
	out := make([]string, 0, len(m.Choices))
	for _, c := range m.Choices {
		out = append(out, strings.TrimSpace(c))
	}
	return out
}

// validateChoice: 3~4 distinct options, exactly one correct (rubric.correct),
// and nothing giving the answer away — the prompt does not contain the correct
// option, and the correct option is not conspicuously longer than the others.
func validateChoice(it Item, label string) []string {
	var p []string
	choices := Choices(it.Material)
	if n := len(choices); n < MinChoices || n > MaxChoices {
		return append(p, fmt.Sprintf("%s: material.choices 는 %d~%d개입니다(현재 %d개)", label, MinChoices, MaxChoices, n))
	}
	seen := map[string]bool{}
	for _, c := range choices {
		key := strings.Join(Normalize(c), " ")
		if key == "" {
			p = append(p, label+": 빈 보기가 있습니다")
		} else if seen[key] {
			p = append(p, fmt.Sprintf("%s: 보기 %q 가 겹칩니다", label, c))
		}
		seen[key] = true
	}
	c := it.Rubric.Correct
	if c == nil || *c < 0 || *c >= len(choices) {
		return append(p, fmt.Sprintf("%s: rubric.correct 는 정답 보기 하나의 번호(0~%d)입니다", label, len(choices)-1))
	}
	correct := choices[*c]
	if ContainsPhrase(it.Prompt, correct) || Leaks(it.Prompt, correct) {
		p = append(p, label+": 질문 문장에 정답 보기가 드러납니다")
	}
	longest := 0
	for i, o := range choices {
		if i != *c {
			longest = max(longest, utf8.RuneCountInString(o))
		}
	}
	if utf8.RuneCountInString(correct) > choiceLengthRatio*longest {
		p = append(p, label+": 정답 보기만 다른 보기보다 두 배 넘게 깁니다")
	}
	return p
}

// ContainsPhrase reports whether text contains phrase word for word after
// Normalize (particles stripped), e.g. "커널 모드에서는" contains "커널 모드".
func ContainsPhrase(text, phrase string) bool {
	want := strings.Join(Normalize(phrase), " ")
	if utf8.RuneCountInString(want) < 2 {
		return false
	}
	return strings.Contains(" "+strings.Join(Normalize(text), " ")+" ", " "+want+" ")
}

// NormalizeDiagnostic prepares a validated diagnostic set for storage: trimmed
// text, an object material, and for choice items trimmed options with
// rubric.answer = the correct option (code decides, not the model).
func NormalizeDiagnostic(set ItemSet) []Item {
	out := make([]Item, 0, len(set.Items))
	for _, it := range set.Items {
		it.Prompt = strings.TrimSpace(it.Prompt)
		it.Rubric.Must = nonEmpty(it.Rubric.Must)
		it.Rubric.Nice = nonEmpty(it.Rubric.Nice)
		it.Rubric.Misconceptions = nonEmpty(it.Rubric.Misconceptions)
		it.Rubric.Answer = strings.TrimSpace(it.Rubric.Answer)
		it.Rubric.Covers = []string{}
		if len(it.Material) == 0 || !isJSONObject(it.Material) {
			it.Material = json.RawMessage(`{}`)
		}
		if it.Format == FormatChoice {
			it.Material = withChoices(it.Material, Choices(it.Material))
			if it.Rubric.Correct != nil {
				it.Rubric.Answer = Choices(it.Material)[*it.Rubric.Correct]
			}
		} else {
			it.Rubric.Correct = nil
		}
		out = append(out, it)
	}
	return out
}

func withChoices(material json.RawMessage, choices []string) json.RawMessage {
	m := map[string]any{}
	if err := json.Unmarshal(material, &m); err != nil {
		m = map[string]any{}
	}
	m["choices"] = choices
	raw, err := json.Marshal(m)
	if err != nil {
		return material
	}
	return raw
}
