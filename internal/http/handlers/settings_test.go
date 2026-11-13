package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSettingsPatchNeedsOneValidField(t *testing.T) {
	ptr := func(n int) *int { return &n }
	str := func(s string) *string { return &s }
	off := false
	cases := []struct {
		name string
		body settingsBody
		code ErrorCode // "" = valid
	}{
		{"empty", settingsBody{}, ErrorSettingsRequired},
		{"daily only", settingsBody{DailyMinutes: ptr(45)}, ""},
		{"mastery target only", settingsBody{MasteryTarget: ptr(5)}, ""},
		{"review grader only", settingsBody{ReviewGrader: str("LLM")}, ""},
		{"intervals only", settingsBody{ReviewIntervals: []int{1, 3, 9}}, ""},
		{"pretest off only", settingsBody{PretestEnabled: &off}, ""},
		{"bad daily", settingsBody{DailyMinutes: ptr(20)}, ErrorInvalidDailyMinutes},
		{"bad target", settingsBody{MasteryTarget: ptr(2)}, ErrorInvalidMasteryTarget},
		{"bad intervals", settingsBody{ReviewIntervals: []int{7, 1}}, ErrorInvalidIntervals},
		{"empty intervals", settingsBody{ReviewIntervals: []int{}}, ErrorInvalidIntervals},
		{"bad minutes", settingsBody{ReviewMinutes: ptr(31)}, ErrorInvalidReviewMinutes},
		{"bad grader", settingsBody{ReviewGrader: str("llm_regrade")}, ErrorInvalidGrader},
		{"bad language", settingsBody{ExampleLanguage: str("cobol")}, ErrorInvalidLanguage},
	}
	for _, c := range cases {
		p, invalid := settingsPatch(c.body)
		var code ErrorCode
		if invalid != nil {
			code = invalid.code
		}
		if code != c.code || (invalid != nil && invalid.msg == "") {
			t.Errorf("%s: code %q, want %q (%+v)", c.name, code, c.code, invalid)
		}
		if c.name == "review grader only" && (p.ReviewGrader == nil || *p.ReviewGrader != "llm") {
			t.Errorf("grader must be normalized: %+v", p.ReviewGrader)
		}
	}
}

// PUT /me/settings answers 400 with the cause's code (the web maps it to Korean).
// Standalone: the user is the fixed local user (no auth context).
func TestPutSettingsBadRequestCode(t *testing.T) {
	h := &LearnHandler{}
	for body, want := range map[string]ErrorCode{
		`{"dailyMinutes":20}`:         ErrorInvalidDailyMinutes,
		`{"reviewIntervals":[7,1]}`:   ErrorInvalidIntervals,
		`{}`:                          ErrorSettingsRequired,
		`{"unknown":1}`:               ErrorBadRequest,
		`{"exampleLanguage":"cobol"}`: ErrorInvalidLanguage,
	} {
		req := httptest.NewRequest(http.MethodPut, "/v1/me/settings", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.PutSettings(rec, req)
		var got errorEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != http.StatusBadRequest || got.Error.Code != want {
			t.Errorf("%s → %d %+v (%v), want 400 %s", body, rec.Code, got.Error, err, want)
		}
	}
}
