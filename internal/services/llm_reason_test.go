package services

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/forgen-team/punisher/internal/i18n"
	"github.com/forgen-team/punisher/internal/llm"
)

// Claude 실패 사유는 모두 코드가 있어야 화면이 언어에 맞게 옮긴다(영어 화면에 한국어 사유가 뜨던 문제).
func TestLLMErrorReasonsAreTranslatable(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{&llm.LLMError{Code: codeCLIMissing}, i18n.ReasonClaudeMissing},
		{&llm.LLMError{Code: codeNotLoggedIn}, i18n.ReasonClaudeLoggedOut},
		{&llm.LLMError{Code: codeRateLimited}, i18n.ReasonClaudeRateLimited},
		{&llm.LLMError{Code: codeQueueFull}, i18n.ReasonClaudeQueueFull},
		{&llm.LLMError{Code: codeDailyCap}, i18n.ReasonClaudeDailyCap},
		{fmt.Errorf("draft: %w", &llm.LLMError{Code: codeTimeout}), i18n.ReasonClaudeTimeout},
		{fmt.Errorf("outline: %w", ErrUnusableOutput), i18n.ReasonClaudeBadOutput},
		{&llm.LLMError{Message: "exit status 1: something the CLI printed"}, i18n.ReasonClaudeFailed},
		{errors.New("not a generation error"), ""},
	}
	for _, c := range cases {
		code := LLMErrorReason(c.err)
		if code != c.want {
			t.Errorf("%v: code %q, want %q", c.err, code, c.want)
			continue
		}
		if code == "" {
			if DescribeLLMError(c.err) != "" {
				t.Errorf("%v: a non-generation error has no text", c.err)
			}
			continue
		}
		// 저장·스트림되는 문구가 코드로 되돌아가야 stage 이벤트에 reasonCodes 가 붙는다.
		if got := i18n.ReasonCodes([]string{llmFailReason(c.err)}); !reflect.DeepEqual(got, []string{code}) {
			t.Errorf("%v: reason codes %v, want [%s]", c.err, got, code)
		}
	}
}
