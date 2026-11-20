package services

import (
	"reflect"
	"strings"
	"testing"

	"github.com/forgen-team/punisher/internal/i18n"
	"github.com/forgen-team/punisher/internal/llm"
)

// The failure text names the AI that failed (learner-level design ⑥).
func TestLLMErrorReasonNamesTheProvider(t *testing.T) {
	cases := []struct {
		err  error
		want string
		name string // must appear in the stored text
	}{
		{&llm.LLMError{Code: "codex_unavailable", Provider: "codex", Err: llm.ErrGenerationUnavailable}, i18n.ReasonCodexMissing, "codex 명령"},
		{&llm.LLMError{Code: "codex_not_logged_in", Provider: "codex"}, i18n.ReasonCodexLoggedOut, "codex login"},
		{&llm.LLMError{Code: "codex_rate_limited", Provider: "codex"}, i18n.ReasonCodexRateLimited, "Codex"},
		{&llm.LLMError{Code: "codex_timeout", Provider: "codex", Err: llm.ErrTimeout}, i18n.ReasonCodexTimeout, "Codex"},
		{&llm.LLMError{Code: codeDailyCap, Provider: "codex"}, i18n.ReasonCodexDailyCap, "Codex"},
		{&llm.LLMError{Code: codeQueueFull, Provider: "codex"}, i18n.ReasonClaudeQueueFull, ""},
		{&llm.LLMError{Code: "codex_error", Provider: "codex", Message: "boom"}, i18n.ReasonCodexFailed, "Codex"},
		{&llm.LLMError{Code: "agy_unavailable", Provider: "agy", Err: llm.ErrGenerationUnavailable}, i18n.ReasonAgyMissing, "agy 명령"},
		{&llm.LLMError{Code: "agy_not_logged_in", Provider: "agy"}, i18n.ReasonAgyLoggedOut, "Antigravity"},
		{&llm.LLMError{Code: "agy_rate_limited", Provider: "agy"}, i18n.ReasonAgyRateLimited, "Antigravity"},
		{&llm.LLMError{Code: "claude_unavailable"}, i18n.ReasonClaudeMissing, "claude 명령"},
	}
	for _, c := range cases {
		code := LLMErrorReason(c.err)
		if code != c.want {
			t.Errorf("%v: code %q, want %q", c.err, code, c.want)
			continue
		}
		text := DescribeLLMError(c.err)
		if !strings.Contains(text, c.name) {
			t.Errorf("%v: text %q does not name %q", c.err, text, c.name)
		}
		if got := i18n.ReasonCodes([]string{text}); !reflect.DeepEqual(got, []string{code}) {
			t.Errorf("%v: reason codes %v, want [%s]", c.err, got, code)
		}
	}
}
