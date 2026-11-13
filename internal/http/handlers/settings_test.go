package handlers

import "testing"

func TestSettingsPatchNeedsOneValidField(t *testing.T) {
	ptr := func(n int) *int { return &n }
	str := func(s string) *string { return &s }
	cases := []struct {
		name string
		body settingsBody
		ok   bool
	}{
		{"empty", settingsBody{}, false},
		{"daily only", settingsBody{DailyMinutes: ptr(45)}, true},
		{"mastery target only", settingsBody{MasteryTarget: ptr(5)}, true},
		{"review grader only", settingsBody{ReviewGrader: str("LLM")}, true},
		{"intervals only", settingsBody{ReviewIntervals: []int{1, 3, 9}}, true},
		{"bad daily", settingsBody{DailyMinutes: ptr(20)}, false},
		{"bad target", settingsBody{MasteryTarget: ptr(2)}, false},
		{"bad intervals", settingsBody{ReviewIntervals: []int{7, 1}}, false},
		{"empty intervals", settingsBody{ReviewIntervals: []int{}}, false},
		{"bad minutes", settingsBody{ReviewMinutes: ptr(31)}, false},
		{"bad grader", settingsBody{ReviewGrader: str("llm_regrade")}, false},
		{"bad language", settingsBody{ExampleLanguage: str("cobol")}, false},
	}
	for _, c := range cases {
		p, msg := settingsPatch(c.body)
		if (msg == "") != c.ok {
			t.Errorf("%s: msg %q", c.name, msg)
		}
		if c.name == "review grader only" && (p.ReviewGrader == nil || *p.ReviewGrader != "llm") {
			t.Errorf("grader must be normalized: %+v", p.ReviewGrader)
		}
	}
}
