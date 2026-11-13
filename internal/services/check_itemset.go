package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wooo-jin/punisher/internal/check"
	"github.com/wooo-jin/punisher/internal/store"
	"github.com/wooo-jin/punisher/internal/llm"
	"github.com/wooo-jin/punisher/internal/llmjson"
)

// writeItemSet runs an item generation with the check.items system prompt:
// one rewrite with the problems when the output does not parse or validate.
// A set that never validates is an error, never items.
func (s *Service) writeItemSet(ctx context.Context, apiKey, system string, priority llm.Priority,
	user func(revisions []string) string, validate func(check.ItemSet) []string) (check.ItemSet, usage, []store.AttemptRecord, error) {
	var u usage
	var tries []store.AttemptRecord
	var problems []string
	for n := 1; n <= checkAttempts; n++ {
		text, gerr := s.complete(ctx, apiKey, llm.MessagesRequest{Model: s.Cfg.LectureModel, System: system,
			User: user(problems), MaxTokens: 8000, Timeout: 5 * time.Minute, Priority: priority}, &u)
		if gerr != nil {
			tries = append(tries, store.AttemptRecord{Attempt: n, Result: "llm_error", Message: gerr.Error()})
			return check.ItemSet{}, u, tries, gerr
		}
		parsed, perr := llmjson.TryParseJSON[check.ItemSet](text)
		if perr != nil {
			problems = []string{"JSON 을 해석할 수 없습니다: " + perr.Error()}
			tries = append(tries, store.AttemptRecord{Attempt: n, Result: "parse_error", Message: perr.Error()})
			continue
		}
		if problems = validate(parsed); len(problems) > 0 {
			tries = append(tries, store.AttemptRecord{Attempt: n, Result: "invalid", Message: strings.Join(problems, "; ")})
			continue
		}
		tries = append(tries, store.AttemptRecord{Attempt: n, Result: "ok"})
		return parsed, u, tries, nil
	}
	return check.ItemSet{}, u, tries, errors.New("check items failed validation: " + strings.Join(problems, "; "))
}
